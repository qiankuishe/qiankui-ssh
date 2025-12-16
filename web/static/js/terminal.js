/**
 * Qiankui SSH - 终端逻辑
 */

// 全局变量
let term = null;
let ws = null;
let fitAddon = null;
let sessionId = null;

// 初始化
document.addEventListener('DOMContentLoaded', function () {
    // 监听表单提交
    document.getElementById('connectForm').addEventListener('submit', handleConnect);

    // 监听断开连接 - 使用事件委托确保全屏后也能响应
    document.addEventListener('click', function (e) {
        if (e.target.closest('#disconnectBtn')) {
            e.preventDefault();
            e.stopPropagation();
            disconnect();
        }
        if (e.target.closest('#fullscreenBtn')) {
            e.preventDefault();
            e.stopPropagation();
            toggleFullscreen();
        }
    });

    // 监听窗口大小变化
    window.addEventListener('resize', debounce(handleResize, 100));

    // 监听 ESC 退出全屏
    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') {
            const container = document.getElementById('terminalContainer');
            if (container && container.classList.contains('fullscreen')) {
                toggleFullscreen();
            }
        }
    });
});

/**
 * 处理连接
 */
async function handleConnect(e) {
    e.preventDefault();

    const formData = new FormData(e.target);
    const data = {
        hostname: formData.get('hostname'),
        port: parseInt(formData.get('port')) || 22,
        username: formData.get('username'),
        password: formData.get('password'),
        privatekey: formData.get('privatekey'),
        passphrase: formData.get('passphrase')
    };

    // 验证
    if (!data.hostname) {
        showStatus('请输入主机地址', 'error');
        return;
    }
    if (!data.username) {
        showStatus('请输入用户名', 'error');
        return;
    }
    if (!data.password && !data.privatekey) {
        showStatus('请输入密码或提供私钥', 'error');
        return;
    }

    // 显示加载
    showLoading(true);

    try {
        // 发送连接请求
        const response = await fetch('/connect', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.success) {
            sessionId = result.session_id;
            showStatus('连接成功', 'success');

            // 切换到终端视图
            showTerminal(data.hostname, data.username);

            // 建立 WebSocket 连接
            connectWebSocket();
        } else {
            showStatus(result.message || '连接失败', 'error');
        }
    } catch (error) {
        showStatus('连接失败: ' + error.message, 'error');
    } finally {
        showLoading(false);
    }
}

/**
 * 建立 WebSocket 连接
 */
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?session_id=${sessionId}`;

    ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = function () {
        console.log('WebSocket 已连接');
        initTerminal();
    };

    ws.onmessage = function (event) {
        if (term) {
            const data = event.data instanceof ArrayBuffer
                ? new TextDecoder().decode(event.data)
                : event.data;
            term.write(data);
        }
    };

    ws.onclose = function (event) {
        console.log('WebSocket 已断开:', event.reason);
        if (term) {
            term.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n');
        }
    };

    ws.onerror = function (error) {
        console.error('WebSocket 错误:', error);
        showStatus('连接错误', 'error');
    };
}

/**
 * 初始化终端
 */
function initTerminal() {
    // 创建终端实例
    term = new Terminal({
        cursorBlink: true,
        cursorStyle: 'bar',
        fontSize: 14,
        fontFamily: '"Cascadia Code", "Fira Code", Consolas, "Courier New", monospace',
        theme: {
            background: '#1e1e1e',
            foreground: '#d4d4d4',
            cursor: '#F4A900',
            cursorAccent: '#1e1e1e',
            selection: 'rgba(244, 169, 0, 0.3)',
            black: '#1e1e1e',
            red: '#f44747',
            green: '#6a9955',
            yellow: '#F4A900',
            blue: '#569cd6',
            magenta: '#c586c0',
            cyan: '#4ec9b0',
            white: '#d4d4d4',
            brightBlack: '#808080',
            brightRed: '#f44747',
            brightGreen: '#6a9955',
            brightYellow: '#F4A900',
            brightBlue: '#569cd6',
            brightMagenta: '#c586c0',
            brightCyan: '#4ec9b0',
            brightWhite: '#ffffff'
        },
        allowTransparency: true,
        scrollback: 10000
    });

    // 加载插件
    fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    // Web Links 插件
    const webLinksAddon = new WebLinksAddon.WebLinksAddon();
    term.loadAddon(webLinksAddon);

    // 挂载到 DOM
    const terminalEl = document.getElementById('terminal');
    term.open(terminalEl);

    // 适配大小
    setTimeout(() => {
        fitAddon.fit();
        sendResize();
    }, 100);

    // 监听输入
    term.onData(function (data) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'data',
                data: data
            }));
        }
    });

    // 聚焦终端
    term.focus();
}

/**
 * 发送终端大小
 */
function sendResize() {
    if (ws && ws.readyState === WebSocket.OPEN && term) {
        ws.send(JSON.stringify({
            type: 'resize',
            resize: {
                cols: term.cols,
                rows: term.rows
            }
        }));
    }
}

/**
 * 处理窗口大小变化
 */
function handleResize() {
    if (fitAddon) {
        fitAddon.fit();
        sendResize();
    }
}

/**
 * 切换全屏
 */
function toggleFullscreen() {
    const container = document.getElementById('terminalContainer');
    if (!container) return;

    const icon = document.querySelector('#fullscreenBtn i');
    container.classList.toggle('fullscreen');

    if (container.classList.contains('fullscreen')) {
        if (icon) {
            icon.classList.remove('fa-expand');
            icon.classList.add('fa-compress');
        }
        // 隐藏GitHub链接
        const githubHeader = document.querySelector('.github-header');
        if (githubHeader) githubHeader.style.display = 'none';
    } else {
        if (icon) {
            icon.classList.remove('fa-compress');
            icon.classList.add('fa-expand');
        }
        // 显示GitHub链接
        const githubHeader = document.querySelector('.github-header');
        if (githubHeader) githubHeader.style.display = 'flex';
    }

    // 重新适配大小
    setTimeout(handleResize, 100);
}

/**
 * 显示终端视图
 */
function showTerminal(hostname, username) {
    document.getElementById('formContainer').style.display = 'none';
    document.getElementById('terminalContainer').style.display = 'block';
    document.getElementById('terminalTitle').textContent = `${username}@${hostname}`;

    // 隐藏背景装饰
    const bgDecoration = document.querySelector('.bg-decoration');
    if (bgDecoration) bgDecoration.style.display = 'none';
}

/**
 * 显示表单视图
 */
function showForm() {
    const formContainer = document.getElementById('formContainer');
    const terminalContainer = document.getElementById('terminalContainer');

    if (formContainer) formContainer.style.display = 'block';
    if (terminalContainer) {
        terminalContainer.style.display = 'none';
        terminalContainer.classList.remove('fullscreen');
    }

    // 显示背景装饰
    const bgDecoration = document.querySelector('.bg-decoration');
    if (bgDecoration) bgDecoration.style.display = 'block';

    // 显示GitHub链接
    const githubHeader = document.querySelector('.github-header');
    if (githubHeader) githubHeader.style.display = 'flex';

    // 重置全屏按钮图标
    const icon = document.querySelector('#fullscreenBtn i');
    if (icon) {
        icon.classList.remove('fa-compress');
        icon.classList.add('fa-expand');
    }
}

/**
 * 断开连接
 */
function disconnect() {
    if (ws) {
        ws.close();
        ws = null;
    }
    if (term) {
        term.dispose();
        term = null;
    }
    fitAddon = null;
    sessionId = null;
    showForm();
    showStatus('已断开连接', 'info');
}

/**
 * 显示加载动画
 */
function showLoading(show) {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) overlay.style.display = show ? 'flex' : 'none';
}

/**
 * 显示状态消息
 */
function showStatus(message, type = 'info') {
    const statusEl = document.getElementById('statusMessage');
    if (!statusEl) return;

    statusEl.textContent = message;
    statusEl.className = 'status-message ' + type;
    statusEl.style.display = 'block';
    setTimeout(() => {
        statusEl.style.display = 'none';
    }, 3000);
}

/**
 * 防抖函数
 */
function debounce(func, wait) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}
