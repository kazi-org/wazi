import AppKit
import UniformTypeIdentifiers
import WebKit

@MainActor
final class DesktopWindowController: NSWindowController, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate, WKScriptMessageHandler {
    var onRestart: (() -> Void)?
    var onQuit: (() -> Void)?
    var onWindowAction: ((DesktopContract.Action) -> Void)?
    var onWebContentFailure: (() -> Void)?

    private var webView: WKWebView?
    private var recoveryController: RecoveryViewController?
    private var pinnedOrigin: URL?
    private var dragEventMonitor: Any?
    private var pendingDragEvent: NSEvent?
    private var pendingDragTime: Date?

    init() {
        let frame = NSRect(x: 0, y: 0, width: 1_180, height: 820)
        let window = KeyableMainWindow(
            contentRect: frame,
            styleMask: [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.minSize = NSSize(width: 400, height: 620)
        window.isOpaque = true
        window.backgroundColor = NSColor(calibratedRed: 0.035, green: 0.047, blue: 0.078, alpha: 1)
        window.hasShadow = true
        window.isMovableByWindowBackground = false
        window.title = "Wazi"
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.titlebarSeparatorStyle = .none
        window.collectionBehavior = [.fullScreenPrimary]
        for button in [NSWindow.ButtonType.closeButton, .miniaturizeButton, .zoomButton] {
            window.standardWindowButton(button)?.isHidden = true
        }
        super.init(window: window)
        window.delegate = self
        installDragMonitor()
    }

    required init?(coder: NSCoder) { nil }

    func showStarting() {
        discardWebView()
        let view = RecoveryViewController(title: "Starting Wazi", message: "Opening the local observatory…", showsRestart: false)
        view.onQuit = { [weak self] in self?.onQuit?() }
        setContent(view.view)
        recoveryController = view
    }

    func showRecovery(title: String = "Wazi stopped", message: String) {
        discardWebView()
        let view = RecoveryViewController(title: title, message: message, showsRestart: true)
        view.onRestart = { [weak self] in self?.onRestart?() }
        view.onQuit = { [weak self] in self?.onQuit?() }
        setContent(view.view)
        recoveryController = view
    }

    func showWeb(at origin: URL) {
        discardWebView()
        recoveryController = nil
        pinnedOrigin = origin
        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = .nonPersistent()
        configuration.preferences.javaScriptCanOpenWindowsAutomatically = false
        let controller = WKUserContentController()
        let metadata = #"Object.defineProperty(window, "__WAZI_DESKTOP__", {value:Object.freeze({platform:"macos",protocol:"wazi-desktop/1"}),writable:false,configurable:false});"#
        controller.addUserScript(WKUserScript(source: metadata, injectionTime: .atDocumentStart, forMainFrameOnly: true, in: .page))
        controller.add(self, name: "waziWindow")
        configuration.userContentController = controller

        let view = WKWebView(frame: .zero, configuration: configuration)
        view.navigationDelegate = self
        view.uiDelegate = self
        view.allowsBackForwardNavigationGestures = false
        view.autoresizingMask = [.width, .height]
        view.setAccessibilityLabel("Wazi observatory")
        webView = view
        setContent(view)
        if let window = view.window {
            window.makeFirstResponder(view)
            window.displayIfNeeded()
        }
        view.load(URLRequest(url: origin.appendingPathComponent("")))
    }

    func windowWillClose(_ notification: Notification) {
        discardWebView()
        recoveryController = nil
        removeDragMonitor()
    }

    static func recoveryLifecycleSelfTest() throws {
        _ = NSApplication.shared
        let controller = DesktopWindowController()
        var restartCount = 0
        var quitCount = 0
        controller.onRestart = { restartCount += 1 }
        controller.onQuit = { quitCount += 1 }

        controller.showRecovery(title: "Wazi stopped", message: "Recovery test")
        weak var replacedRecovery: RecoveryViewController? = controller.recoveryController
        guard invokeButtonAction(title: "Restart Wazi", in: controller.window?.contentView),
              restartCount == 1,
              invokeButtonAction(title: "Quit Wazi", in: controller.window?.contentView),
              quitCount == 1 else { throw RecoveryTestError.actionNotDelivered }

        controller.showRecovery(title: "Wazi stopped", message: "Replacement test")
        guard replacedRecovery == nil,
              invokeButtonAction(title: "Restart Wazi", in: controller.window?.contentView),
              restartCount == 2,
              invokeButtonAction(title: "Quit Wazi", in: controller.window?.contentView),
              quitCount == 2 else { throw RecoveryTestError.replacementFailed }

        weak var closingRecovery: RecoveryViewController? = controller.recoveryController
        controller.windowWillClose(Notification(name: NSWindow.willCloseNotification, object: controller.window))
        guard closingRecovery == nil else { throw RecoveryTestError.closeDidNotRelease }
    }

    private static func invokeButtonAction(title: String, in root: NSView?) -> Bool {
        guard let root else { return false }
        if let button = root as? NSButton, button.title == title {
            guard let action = button.action, let target = button.target else { return false }
            return NSApplication.shared.sendAction(action, to: target, from: button)
        }
        return root.subviews.contains { invokeButtonAction(title: title, in: $0) }
    }

    private enum RecoveryTestError: Error {
        case actionNotDelivered
        case replacementFailed
        case closeDidNotRelease
    }

    func windowDidBecomeKey(_ notification: Notification) {
        webView?.window?.makeFirstResponder(webView)
    }

    func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
        guard self.webView === webView else { return }
        onWebContentFailure?()
    }

    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard let sourceWebView = message.webView,
              self.webView === sourceWebView,
              sourceWebView.configuration.userContentController === userContentController,
              message.frameInfo.isMainFrame,
              let origin = pinnedOrigin,
              DesktopContract.matches(origin, scheme: message.frameInfo.securityOrigin.`protocol`, host: message.frameInfo.securityOrigin.host, port: message.frameInfo.securityOrigin.port),
              let action = DesktopContract.action(from: message.body)
        else { return }

        if action == .drag {
            beginWindowDragIfCurrentEvent()
        } else {
            onWindowAction?(action)
        }
    }

    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        guard self.webView === webView,
              let origin = pinnedOrigin,
              let url = navigationAction.request.url
        else { decisionHandler(.cancel); return }

        if let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
           components.user == nil, components.password == nil,
           DesktopContract.matches(origin, scheme: components.scheme ?? "", host: components.host ?? "", port: components.port ?? -1),
           navigationAction.targetFrame != nil {
            decisionHandler(.allow)
            return
        }

        if navigationAction.navigationType == .linkActivated,
           navigationAction.sourceFrame.isMainFrame,
           let external = DesktopContract.validatedExternalHTTPURL(url),
           !DesktopContract.matches(origin, scheme: URLComponents(url: external, resolvingAgainstBaseURL: false)?.scheme ?? "", host: URLComponents(url: external, resolvingAgainstBaseURL: false)?.host ?? "", port: URLComponents(url: external, resolvingAgainstBaseURL: false)?.port ?? -1) {
            NSWorkspace.shared.open(external)
        }
        decisionHandler(.cancel)
    }

    func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
        // target=_blank never creates another privileged or unprivileged web view.
        nil
    }

    func webView(_ webView: WKWebView, runOpenPanelWith parameters: WKOpenPanelParameters, initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping ([URL]?) -> Void) {
        guard self.webView === webView,
              frame.isMainFrame,
              let origin = pinnedOrigin,
              DesktopContract.matches(origin, scheme: frame.securityOrigin.`protocol`, host: frame.securityOrigin.host, port: frame.securityOrigin.port)
        else { completionHandler(nil); return }

        let panel = NSOpenPanel()
        panel.title = "Import a Markdown or JSON plan"
        panel.prompt = "Import"
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.canCreateDirectories = false
        panel.allowsMultipleSelection = false
        panel.treatsFilePackagesAsDirectories = false
        var types: [UTType] = [.json]
        if let markdown = UTType(filenameExtension: "md") { types.append(markdown) }
        panel.allowedContentTypes = types
        panel.begin { response in
            guard response == .OK,
                  let url = panel.url,
                  ["md", "json"].contains(url.pathExtension.lowercased())
            else { completionHandler(nil); return }
            completionHandler([url])
        }
    }

    private func setContent(_ view: NSView) {
        guard let window else { return }
        view.autoresizingMask = [.width, .height]
        window.contentView = view
        view.frame = window.contentView?.bounds ?? window.contentLayoutRect
    }

    private func discardWebView() {
        guard let old = webView else { pinnedOrigin = nil; return }
        old.stopLoading()
        old.navigationDelegate = nil
        old.uiDelegate = nil
        old.configuration.userContentController.removeScriptMessageHandler(forName: "waziWindow")
        old.removeFromSuperview()
        webView = nil
        pinnedOrigin = nil
        pendingDragEvent = nil
        pendingDragTime = nil
    }

    private func installDragMonitor() {
        guard dragEventMonitor == nil else { return }
        dragEventMonitor = NSEvent.addLocalMonitorForEvents(matching: [.leftMouseDown, .leftMouseUp]) { [weak self] event in
            guard let self, event.window === self.window else { return event }
            if event.type == .leftMouseUp {
                self.pendingDragEvent = nil
                self.pendingDragTime = nil
            } else {
                self.recordDragMouseDown(event)
            }
            return event
        }
    }

    private func removeDragMonitor() {
        if let dragEventMonitor { NSEvent.removeMonitor(dragEventMonitor) }
        dragEventMonitor = nil
        pendingDragEvent = nil
        pendingDragTime = nil
    }

    private func recordDragMouseDown(_ event: NSEvent) {
        guard let webView, event.buttonNumber == 0 else { pendingDragEvent = nil; return }
        let point = webView.convert(event.locationInWindow, from: nil)
        guard DesktopContract.isReservedDragPoint(x: point.x, y: point.y, viewHeight: webView.bounds.height, isFlipped: webView.isFlipped) else {
            pendingDragEvent = nil
            pendingDragTime = nil
            return
        }
        pendingDragEvent = event
        pendingDragTime = Date()
    }

    private func beginWindowDragIfCurrentEvent() {
        guard NSEvent.pressedMouseButtons & 1 == 1,
              let event = pendingDragEvent,
              event.window === window,
              let time = pendingDragTime,
              Date().timeIntervalSince(time) <= 0.5,
              let window
        else { pendingDragEvent = nil; pendingDragTime = nil; return }
        pendingDragEvent = nil
        pendingDragTime = nil
        window.performDrag(with: event)
    }
}

@MainActor
final class KeyableMainWindow: NSWindow {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

@MainActor
private final class RecoveryViewController: NSViewController {
    var onRestart: (() -> Void)?
    var onQuit: (() -> Void)?

    private let titleText: String
    private let messageText: String
    private let showsRestart: Bool

    init(title: String, message: String, showsRestart: Bool) {
        self.titleText = title
        self.messageText = message
        self.showsRestart = showsRestart
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { nil }

    override func loadView() {
        let root = NSView()
        root.wantsLayer = true
        root.layer?.backgroundColor = NSColor(calibratedRed: 0.035, green: 0.047, blue: 0.078, alpha: 1).cgColor

        let title = NSTextField(labelWithString: titleText)
        title.font = .systemFont(ofSize: 22, weight: .semibold)
        title.textColor = .white
        title.alignment = .center
        title.setAccessibilityLabel(titleText)

        let message = NSTextField(wrappingLabelWithString: messageText)
        message.font = .systemFont(ofSize: 14)
        message.textColor = NSColor(white: 0.78, alpha: 1)
        message.alignment = .center
        message.maximumNumberOfLines = 3

        let restart = NSButton(title: "Restart Wazi", target: self, action: #selector(restartPressed))
        restart.keyEquivalent = "\r"
        restart.isHidden = !showsRestart
        restart.setAccessibilityLabel("Restart Wazi explicitly")

        let quit = NSButton(title: "Quit Wazi", target: self, action: #selector(quitPressed))
        quit.setAccessibilityLabel("Quit Wazi")

        let stack = NSStackView(views: [title, message, restart, quit])
        stack.orientation = .vertical
        stack.alignment = .centerX
        stack.distribution = .fill
        stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        root.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: root.centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: root.centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: root.leadingAnchor, constant: 28),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: root.trailingAnchor, constant: -28),
            message.widthAnchor.constraint(lessThanOrEqualToConstant: 520)
        ])
        view = root
    }

    @objc private func restartPressed() { onRestart?() }
    @objc private func quitPressed() { onQuit?() }
}
