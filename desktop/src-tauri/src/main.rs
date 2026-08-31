#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;

use tauri::menu::{Menu, MenuItem, PredefinedMenuItem, Submenu};
use tauri::webview::PageLoadEvent;
use tauri::{AppHandle, Manager, RunEvent, WindowEvent};
use tauri_plugin_dialog::{DialogExt, MessageDialogKind};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

const INJECT_OPEN_REPO: &str = r#"
(function () {
  var el = document.getElementById("repo-path");
  if (!el || el.dataset.cmNative === "1") return;
  el.dataset.cmNative = "1";
  el.title = "Open a git repository";
  el.addEventListener("click", function (e) {
    var internals = window.__TAURI_INTERNALS__;
    var invoke = internals && internals.invoke;
    if (!invoke && window.__TAURI__ && window.__TAURI__.core) {
      invoke = window.__TAURI__.core.invoke;
    }
    if (!invoke) return;
    e.preventDefault();
    e.stopImmediatePropagation();
    invoke("open_repo");
  }, true);
})();
"#;

struct Engine {
    child: Mutex<Option<CommandChild>>,
    url: Mutex<Option<String>>,
}

fn main() {
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(Engine {
            child: Mutex::new(None),
            url: Mutex::new(None),
        })
        .invoke_handler(tauri::generate_handler![open_repo])
        .menu(|handle| {
            let open = MenuItem::with_id(
                handle,
                "open-repo",
                "Open repo…",
                true,
                Some("CmdOrCtrl+O"),
            )?;
            let file = Submenu::with_items(
                handle,
                "File",
                true,
                &[
                    &open,
                    &PredefinedMenuItem::separator(handle)?,
                    &PredefinedMenuItem::quit(handle, None)?,
                ],
            )?;
            Menu::with_items(handle, &[&file])
        })
        .on_menu_event(|app, event| {
            if event.id() == "open-repo" {
                prompt_open_repo(app.clone());
            }
        })
        .on_page_load(|webview, payload| {
            if payload.event() == PageLoadEvent::Finished {
                let url = payload.url().as_str();
                if url.contains("127.0.0.1") || url.contains("localhost") {
                    let _ = webview.eval(INJECT_OPEN_REPO);
                }
            }
        })
        .on_window_event(|window, event| {
            if matches!(
                event,
                WindowEvent::Destroyed | WindowEvent::CloseRequested { .. }
            ) {
                kill_engine(window.app_handle());
            }
        })
        .setup(|app| {
            if let Some(w) = app.get_webview_window("main") {
                // None = follow the OS. Title bar and prefers-color-scheme stay in sync.
                let _ = w.set_theme(None);
            }
            if let Some(repo) = load_last_repo(app.handle()) {
                start_engine(app.handle(), Some(repo));
            } else {
                set_status(
                    app.handle(),
                    "Open a git repository — File → Open repo",
                );
            }
            Ok(())
        });

    let app = builder
        .build(tauri::generate_context!())
        .expect("error while building Change Monitor");

    app.run(|app, event| {
        if matches!(event, RunEvent::Exit | RunEvent::ExitRequested { .. }) {
            kill_engine(app);
        }
    });
}

#[tauri::command]
fn open_repo(app: AppHandle) -> Result<(), String> {
    // Commands run off the UI thread, so the blocking picker is safe here.
    let folder = app
        .dialog()
        .file()
        .set_title("Open git repository")
        .blocking_pick_folder();
    let Some(folder) = folder else {
        return Ok(());
    };
    bind_repo(&app, &file_path_string(folder))
}

fn prompt_open_repo(app: AppHandle) {
    // Menu handlers run on the UI thread — use the async picker.
    app.dialog()
        .file()
        .set_title("Open git repository")
        .pick_folder(move |folder| {
            let Some(folder) = folder else { return };
            let path = file_path_string(folder);
            if let Err(err) = bind_repo(&app, &path) {
                show_error(&app, &err);
            }
        });
}

fn file_path_string(folder: tauri_plugin_dialog::FilePath) -> String {
    match folder.into_path() {
        Ok(p) => p.to_string_lossy().into_owned(),
        Err(e) => e.to_string(),
    }
}

fn bind_repo(app: &AppHandle, path: &str) -> Result<(), String> {
    let path = path.trim();
    if path.is_empty() {
        return Err("path required".into());
    }
    save_last_repo(app, path);
    let existing = {
        let engine = app.state::<Engine>();
        let url = engine.url.lock().unwrap().clone();
        url
    };
    if let Some(url) = existing {
        post_repo(&url, path)?;
        if let Some(w) = app.get_webview_window("main") {
            let _ = w.eval("typeof load === 'function' && load()");
        }
        return Ok(());
    }
    start_engine(app, Some(path.to_string()));
    Ok(())
}

fn start_engine(app: &AppHandle, repo: Option<String>) {
    kill_engine(app);
    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        if let Err(err) = run_engine(app.clone(), repo).await {
            set_status(&app, &format!("Engine failed. File → Open repo. {err}"));
            show_error(&app, &err);
        }
    });
}

async fn run_engine(app: AppHandle, repo: Option<String>) -> Result<(), String> {
    set_status(&app, "Starting engine…");
    let port = pick_port()?;
    let addr = format!("127.0.0.1:{port}");
    let mut args = vec!["serve".to_string(), "--addr".to_string(), addr.clone()];
    if let Some(repo) = repo.as_ref() {
        args.push("--repo".to_string());
        args.push(repo.clone());
    }

    let sidecar = app
        .shell()
        .sidecar("aichange")
        .map_err(|e| {
            format!(
                "aichange sidecar missing ({e}). From desktop/: python3 scripts/build-sidecar.py"
            )
        })?;
    let sidecar = with_git_on_path(sidecar);
    let (mut rx, child) = sidecar
        .args(args)
        .spawn()
        .map_err(|e| format!("spawn aichange: {e}"))?;

    {
        let engine = app.state::<Engine>();
        *engine.child.lock().unwrap() = Some(child);
    }

    let url = format!("http://{addr}");
    let mut err_buf = String::new();

    while let Some(event) = rx.recv().await {
        match event {
            CommandEvent::Stdout(line) => {
                let line = String::from_utf8_lossy(line.as_ref());
                if line.contains("aichange listening on") {
                    {
                        let engine = app.state::<Engine>();
                        *engine.url.lock().unwrap() = Some(url.clone());
                    }
                    navigate(&app, &url);
                    return Ok(());
                }
            }
            CommandEvent::Stderr(line) => {
                err_buf.push_str(&String::from_utf8_lossy(line.as_ref()));
                err_buf.push('\n');
            }
            CommandEvent::Terminated(payload) => {
                let msg = err_buf.trim();
                return Err(if msg.is_empty() {
                    format!("engine exited (status {:?})", payload.code)
                } else {
                    msg.to_string()
                });
            }
            CommandEvent::Error(e) => return Err(e),
            _ => {}
        }
    }

    let msg = err_buf.trim();
    Err(if msg.is_empty() {
        "engine stopped before it printed a listening address".into()
    } else {
        msg.to_string()
    })
}

fn kill_engine(app: &AppHandle) {
    if let Some(engine) = app.try_state::<Engine>() {
        if let Some(child) = engine.child.lock().unwrap().take() {
            let _ = child.kill();
        }
        *engine.url.lock().unwrap() = None;
    }
}

fn navigate(app: &AppHandle, url: &str) {
    if let Some(w) = app.get_webview_window("main") {
        let js = format!(
            "window.location.replace({})",
            serde_json::to_string(url).unwrap()
        );
        let _ = w.eval(js);
    }
}

fn set_status(app: &AppHandle, msg: &str) {
    if let Some(w) = app.get_webview_window("main") {
        let js = format!(
            "(function(){{var e=document.getElementById('status');if(e)e.textContent={};}})()",
            serde_json::to_string(msg).unwrap()
        );
        let _ = w.eval(js);
    }
}

fn show_error(app: &AppHandle, msg: &str) {
    app.dialog()
        .message(msg)
        .title("Change Monitor")
        .kind(MessageDialogKind::Error)
        .show(|_| {});
}

fn pick_port() -> Result<u16, String> {
    for port in 7380..7480 {
        if std::net::TcpListener::bind(("127.0.0.1", port)).is_ok() {
            return Ok(port);
        }
    }
    let listener =
        std::net::TcpListener::bind(("127.0.0.1", 0)).map_err(|e| format!("bind: {e}"))?;
    listener
        .local_addr()
        .map(|a| a.port())
        .map_err(|e| format!("local_addr: {e}"))
}

fn post_repo(base: &str, path: &str) -> Result<(), String> {
    let body = serde_json::json!({ "path": path }).to_string();
    let url = format!("{}/api/repo", base.trim_end_matches('/'));
    let rest = url
        .strip_prefix("http://")
        .ok_or_else(|| format!("bad engine url {url}"))?;
    let (hostport, rel) = rest.split_once('/').unwrap_or((rest, ""));
    let request_path = format!("/{rel}");
    let mut stream = std::net::TcpStream::connect(hostport)
        .map_err(|e| format!("connect {hostport}: {e}"))?;
    stream
        .set_read_timeout(Some(Duration::from_secs(30)))
        .ok();
    stream
        .set_write_timeout(Some(Duration::from_secs(15)))
        .ok();
    let req = format!(
        "POST {request_path} HTTP/1.1\r\nHost: {hostport}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    );
    stream
        .write_all(req.as_bytes())
        .map_err(|e| format!("write: {e}"))?;
    let mut resp = String::new();
    stream
        .read_to_string(&mut resp)
        .map_err(|e| format!("read: {e}"))?;
    let status_line = resp.lines().next().unwrap_or("");
    let ok = status_line.contains(" 200 ")
        || status_line.ends_with(" 200")
        || status_line.contains("200 OK");
    if ok {
        return Ok(());
    }
    let body_msg = resp
        .split("\r\n\r\n")
        .nth(1)
        .or_else(|| resp.split("\n\n").nth(1))
        .unwrap_or(status_line)
        .trim();
    Err(if body_msg.is_empty() {
        status_line.to_string()
    } else {
        body_msg.to_string()
    })
}


fn with_git_on_path(cmd: tauri_plugin_shell::process::Command) -> tauri_plugin_shell::process::Command {
    let extras = git_bin_dirs();
    if extras.is_empty() {
        return cmd;
    }
    let sep = if cfg!(windows) { ";" } else { ":" };
    let mut path = extras.join(sep);
    if let Ok(existing) = std::env::var("PATH") {
        path = format!("{path}{sep}{existing}");
    }
    cmd.env("PATH", path)
}

fn git_bin_dirs() -> Vec<String> {
    let mut dirs = Vec::new();
    #[cfg(windows)]
    {
        let pf = std::env::var("ProgramFiles").unwrap_or_else(|_| r"C:\Program Files".into());
        let pf86 =
            std::env::var("ProgramFiles(x86)").unwrap_or_else(|_| r"C:\Program Files (x86)".into());
        for d in [
            format!(r"{pf}\Git\cmd"),
            format!(r"{pf}\Git\bin"),
            format!(r"{pf86}\Git\cmd"),
        ] {
            if Path::new(&d).is_dir() {
                dirs.push(d);
            }
        }
    }
    dirs
}

fn last_repo_file(app: &AppHandle) -> Option<PathBuf> {
    app.path().app_data_dir().ok().map(|d| d.join("last-repo.txt"))
}

fn load_last_repo(app: &AppHandle) -> Option<String> {
    let file = last_repo_file(app)?;
    let s = std::fs::read_to_string(file).ok()?;
    let s = s.trim().to_string();
    if s.is_empty() || !Path::new(&s).is_dir() {
        None
    } else {
        Some(s)
    }
}

fn save_last_repo(app: &AppHandle, path: &str) {
    if let Some(file) = last_repo_file(app) {
        if let Some(dir) = file.parent() {
            let _ = std::fs::create_dir_all(dir);
        }
        let _ = std::fs::write(file, path);
    }
}
