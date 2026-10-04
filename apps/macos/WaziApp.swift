import AppKit
import Darwin
import Foundation

@main
struct WaziDesktopApplication {
    @MainActor
    static func main() {
        if CommandLine.arguments.contains("--self-test") {
            do {
                try DesktopContract.selfTest()
                print("Wazi native contract checks passed")
                exit(EXIT_SUCCESS)
            } catch {
                fputs("Wazi native contract check failed: \(error)\n", stderr)
                exit(EXIT_FAILURE)
            }
        }

        if CommandLine.arguments.contains("--supervisor-test") {
            do {
                try DesktopSupervisorHarness.run()
                print("Wazi supervisor lifecycle checks passed")
                exit(EXIT_SUCCESS)
            } catch {
                fputs("Wazi supervisor lifecycle check failed\n", stderr)
                exit(EXIT_FAILURE)
            }
        }

        let application = NSApplication.shared
        application.setActivationPolicy(.regular)
        let delegate = WaziAppDelegate()
        application.delegate = delegate
        application.run()
    }
}

@MainActor
final class WaziAppDelegate: NSObject, NSApplicationDelegate {
    private var instanceGuard: ScopedInstanceGuard?
    private var windowController: DesktopWindowController?
    private var hostSupervisor: HostSupervisor?
    private var isTerminating = false

    func applicationDidFinishLaunching(_ notification: Notification) {
        installMainMenu()
        do {
            let dataDirectory = try DesktopResources.resolveDataDirectory()
            guard let guardToken = try ScopedInstanceGuard.acquire(dataDirectory: dataDirectory) else {
                activateExistingInstance()
                NSApp.terminate(nil)
                return
            }
            instanceGuard = guardToken
        } catch {
            showInstanceFailure()
            return
        }

        let controller = DesktopWindowController()
        controller.onRestart = { [weak self] in self?.restartSession() }
        controller.onQuit = { NSApp.terminate(nil) }
        controller.onWindowAction = { [weak self] action in self?.performWindowAction(action) }
        controller.onWebContentFailure = { [weak self] in self?.webContentFailed() }
        controller.showStarting()
        controller.showWindow(nil)
        controller.window?.center()
        controller.window?.makeKeyAndOrderFront(nil)
        windowController = controller
        startSession()
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        guard !isTerminating else { return .terminateLater }
        isTerminating = true
        guard let hostSupervisor else { return .terminateNow }
        hostSupervisor.stop { NSApp.reply(toApplicationShouldTerminate: true) }
        return .terminateLater
    }

    private func startSession() {
        guard let windowController else { return }
        windowController.showStarting()
        do {
            let resources = try DesktopResources.resolve()
            let supervisor = HostSupervisor(resources: resources)
            supervisor.onReady = { [weak self] origin in self?.windowController?.showWeb(at: origin) }
            supervisor.onFailure = { [weak self] in
                self?.windowController?.showRecovery(message: "The local observatory stopped. Temporary page state was cleared; no request was replayed.")
            }
            hostSupervisor = supervisor
            supervisor.start()
        } catch {
            windowController.showRecovery(title: "Wazi could not start", message: "A required bundled app resource is unavailable. No project data was opened.")
        }
    }

    private func restartSession() {
        guard !isTerminating, let windowController else { return }
        windowController.showStarting()
        if let hostSupervisor {
            hostSupervisor.stop { [weak self] in self?.startSession() }
        } else {
            startSession()
        }
    }

    private func webContentFailed() {
        windowController?.showRecovery(message: "The web content stopped. Temporary page state was cleared; no request was replayed.")
        hostSupervisor?.stop(completion: {})
    }

    private func performWindowAction(_ action: DesktopContract.Action) {
        guard let window = windowController?.window else { return }
        switch action {
        case .close: window.performClose(nil)
        case .minimize: window.miniaturize(nil)
        case .fullscreen: window.toggleFullScreen(nil)
        case .drag: break // drag requires its own validated current native mouse event
        }
    }

    private func activateExistingInstance() {
        guard let bundleID = Bundle.main.bundleIdentifier else { return }
        let existing = NSRunningApplication.runningApplications(withBundleIdentifier: bundleID)
            .first { $0.processIdentifier != getpid() }
        existing?.activate(options: [.activateAllWindows, .activateIgnoringOtherApps])
    }

    private func showInstanceFailure() {
        let controller = DesktopWindowController()
        controller.onQuit = { NSApp.terminate(nil) }
        controller.onRestart = { [weak self] in self?.retryInstanceGuard() }
        controller.showRecovery(title: "Wazi could not start", message: "The app could not establish its private single-instance guard. No local host was started.")
        controller.showWindow(nil)
        controller.window?.center()
        controller.window?.makeKeyAndOrderFront(nil)
        windowController = controller
    }

    private func retryInstanceGuard() {
        do {
            let dataDirectory = try DesktopResources.resolveDataDirectory()
            guard let guardToken = try ScopedInstanceGuard.acquire(dataDirectory: dataDirectory) else {
                activateExistingInstance()
                return
            }
            instanceGuard = guardToken
            startSession()
        } catch {
            windowController?.showRecovery(title: "Wazi could not start", message: "The app could not establish its private single-instance guard. No local host was started.")
        }
    }

    private func installMainMenu() {
        let main = NSMenu()

        let appRoot = NSMenuItem()
        let appMenu = NSMenu(title: "Wazi")
        appMenu.addItem(NSMenuItem(title: "Quit Wazi", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        appRoot.submenu = appMenu
        main.addItem(appRoot)

        let editRoot = NSMenuItem()
        let edit = NSMenu(title: "Edit")
        edit.addItem(NSMenuItem(title: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c"))
        edit.addItem(NSMenuItem(title: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v"))
        edit.addItem(NSMenuItem(title: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a"))
        editRoot.submenu = edit
        main.addItem(editRoot)

        let windowRoot = NSMenuItem()
        let windowMenu = NSMenu(title: "Window")
        windowMenu.addItem(NSMenuItem(title: "Close Window", action: #selector(NSWindow.performClose(_:)), keyEquivalent: "w"))
        windowMenu.addItem(NSMenuItem(title: "Minimize", action: #selector(NSWindow.miniaturize(_:)), keyEquivalent: "m"))
        windowMenu.addItem(NSMenuItem(title: "Full Screen", action: #selector(NSWindow.toggleFullScreen(_:)), keyEquivalent: "f"))
        windowRoot.submenu = windowMenu
        main.addItem(windowRoot)

        NSApp.mainMenu = main
        NSApp.windowsMenu = windowMenu
    }
}
