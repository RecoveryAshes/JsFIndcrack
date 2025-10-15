# -*- mode: python ; coding: utf-8 -*-


a = Analysis(
    ['main.py'],
    pathex=[],
    binaries=[],
    datas=[('src', 'src')],
    hiddenimports=['src.core.js_crawler', 'src.crawlers.playwright_crawler', 'src.utils.parallel_similarity_analyzer', 'playwright', 'playwright.sync_api', 'playwright._impl._driver', 'playwright._impl._api_structures', 'playwright._impl._browser_type', 'playwright._impl._page', 'playwright._impl._element_handle', 'playwright._impl._frame', 'playwright._impl._network', 'playwright._impl._js_handle', 'playwright._impl._download', 'playwright._impl._file_chooser', 'playwright._impl._video', 'playwright._impl._worker', 'playwright._impl._locator', 'playwright._impl._browser_context', 'playwright._impl._browser', 'playwright._impl._connection', 'playwright._impl._transport', 'playwright._impl._helper', 'playwright._impl._wait_helper', 'playwright._impl._errors', 'playwright._impl._str_utils', 'playwright._impl._api_types', 'playwright._impl._sync_base', 'playwright._impl._async_base', 'playwright._impl._cdp_session', 'playwright._impl._console_message', 'playwright._impl._dialog', 'playwright._impl._keyboard', 'playwright._impl._mouse', 'playwright._impl._touchscreen', 'playwright._impl._accessibility', 'playwright._impl._selectors', 'playwright._impl._tracing', 'playwright._impl._har', 'playwright._impl._clock', 'playwright._impl._route', 'playwright._impl._request', 'playwright._impl._response', 'playwright._impl._web_socket', 'playwright._impl._artifact', 'playwright._impl._fetch', 'playwright._impl._local_utils', 'playwright._impl._glob', 'playwright._impl._zip_file', 'playwright._impl._stream', 'playwright._impl._writeable_stream', 'playwright._impl._readable_stream', 'playwright._impl._event_context_manager', 'playwright._impl._waiter', 'playwright._impl._greenlet_utils', 'playwright._impl._sync_playwright', 'playwright._impl._async_playwright', 'playwright._impl._impl_to_api_mapping', 'playwright._impl._object_factory', 'playwright._impl._path_utils', 'playwright._impl._env_vars', 'playwright._impl._utils', 'playwright._impl._api_names', 'playwright._impl._channels', 'playwright._impl._connection_error', 'playwright._impl._debug_controller', 'playwright._impl._debug_controller_channels', 'playwright._impl._debug_controller_connection', 'playwright._impl._debug_controller_transport', 'playwright._impl._debug_controller_types', 'playwright._impl._debug_controller_utils', 'playwright._impl._debug_controller_waiter', 'playwright._impl._debug_controller_worker', 'playwright._impl._debug_controller_locator', 'playwright._impl._debug_controller_browser_context', 'playwright._impl._debug_controller_browser', 'playwright._impl._debug_controller_connection', 'playwright._impl._debug_controller_transport', 'playwright._impl._debug_controller_helper', 'playwright._impl._debug_controller_wait_helper', 'playwright._impl._debug_controller_errors', 'playwright._impl._debug_controller_str_utils', 'playwright._impl._debug_controller_api_types', 'playwright._impl._debug_controller_sync_base', 'playwright._impl._debug_controller_async_base', 'playwright._impl._debug_controller_cdp_session', 'playwright._impl._debug_controller_console_message', 'playwright._impl._debug_controller_dialog', 'playwright._impl._debug_controller_keyboard', 'playwright._impl._debug_controller_mouse', 'playwright._impl._debug_controller_touchscreen', 'playwright._impl._debug_controller_accessibility', 'playwright._impl._debug_controller_selectors', 'playwright._impl._debug_controller_tracing', 'playwright._impl._debug_controller_har', 'playwright._impl._debug_controller_clock', 'playwright._impl._debug_controller_route', 'playwright._impl._debug_controller_request', 'playwright._impl._debug_controller_response', 'playwright._impl._debug_controller_web_socket', 'playwright._impl._debug_controller_artifact', 'playwright._impl._debug_controller_fetch', 'playwright._impl._debug_controller_local_utils', 'playwright._impl._debug_controller_glob', 'playwright._impl._debug_controller_zip_file', 'playwright._impl._debug_controller_stream', 'playwright._impl._debug_controller_writeable_stream', 'playwright._impl._debug_controller_readable_stream', 'playwright._impl._debug_controller_event_context_manager', 'playwright._impl._debug_controller_waiter', 'playwright._impl._debug_controller_greenlet_utils', 'playwright._impl._debug_controller_sync_playwright', 'playwright._impl._debug_controller_async_playwright', 'playwright._impl._debug_controller_impl_to_api_mapping', 'playwright._impl._debug_controller_object_factory', 'playwright._impl._debug_controller_path_utils', 'playwright._impl._debug_controller_env_vars', 'playwright._impl._debug_controller_utils', 'playwright._impl._debug_controller_api_names', 'playwright._impl._debug_controller_channels', 'playwright._impl._debug_controller_connection_error'],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    noarchive=False,
    optimize=0,
)
pyz = PYZ(a.pure)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='jsfindcrack-macos-arm64',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
