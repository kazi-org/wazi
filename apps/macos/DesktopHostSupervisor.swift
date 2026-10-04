import AppKit
import Darwin
import Foundation

struct DesktopResources {
    let host: URL
    let node: URL
    let bridge: URL
    let root: URL
    let data: URL
    let web: URL

    static func resolve(bundle: Bundle = .main, environment: [String: String] = ProcessInfo.processInfo.environment) throws -> DesktopResources {
        guard let resources = bundle.resourceURL else { throw ResourceError.missing }
        let host = resources.appendingPathComponent("host/wazi", isDirectory: false)
        let node = resources.appendingPathComponent("runtime/node", isDirectory: false)
        let bridge = resources.appendingPathComponent("parser/scripts/host-bridge.mjs", isDirectory: false)
        let web = resources.appendingPathComponent("web", isDirectory: true)
        let home = URL(fileURLWithPath: NSHomeDirectory(), isDirectory: true)
        let root = try configuredDirectory(environment["WAZI_DESKTOP_ROOT"], fallback: home.appendingPathComponent("Code", isDirectory: true))
        let data = try resolveDataDirectory(environment: environment)

        let manager = FileManager.default
        guard manager.isExecutableFile(atPath: host.path),
              manager.isExecutableFile(atPath: node.path),
              manager.fileExists(atPath: bridge.path),
              manager.fileExists(atPath: resources.appendingPathComponent("parser/scripts/plans.mjs").path),
              manager.fileExists(atPath: resources.appendingPathComponent("parser/src/plan-parser.mjs").path),
              manager.fileExists(atPath: web.appendingPathComponent("index.html").path)
        else { throw ResourceError.missing }
        return DesktopResources(host: host, node: node, bridge: bridge, root: root, data: data, web: web)
    }

    static func resolveDataDirectory(environment: [String: String] = ProcessInfo.processInfo.environment) throws -> URL {
        let fallback = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Application Support/Wazi", isDirectory: true)
        return try configuredDirectory(environment["WAZI_DESKTOP_DATA"], fallback: fallback)
    }

    private static func configuredDirectory(_ value: String?, fallback: URL) throws -> URL {
        guard let value, !value.isEmpty else { return fallback.standardizedFileURL }
        let expanded = (value as NSString).expandingTildeInPath
        guard expanded.hasPrefix("/") else { throw ResourceError.invalidPath }
        return URL(fileURLWithPath: expanded, isDirectory: true).standardizedFileURL
    }

    enum ResourceError: Error {
        case missing
        case invalidPath
    }
}

@MainActor
final class HostSupervisor {
    private enum State: Equatable { case idle, starting, ready, stopping }

    var onReady: ((URL) -> Void)?
    var onFailure: (() -> Void)?

    private let resources: DesktopResources
    private let startupTimeout: TimeInterval
    private let shutdownGrace: TimeInterval
    private var state: State = .idle
    private var process: Process?
    private var stdinPipe: Pipe?
    private var stdoutPipe: Pipe?
    private var stderrPipe: Pipe?
    private var nonce = ""
    private var launchID = UUID()
    private var stdoutBuffer = Data()
    private var startupTimer: DispatchWorkItem?
    private var shutdownTimer: DispatchWorkItem?
    private var stopCompletions: [() -> Void] = []

    var hasOwnedProcess: Bool { process != nil }

    init(
        resources: DesktopResources,
        startupTimeout: TimeInterval = DesktopContract.startupTimeout,
        shutdownGrace: TimeInterval = DesktopContract.shutdownGrace
    ) {
        self.resources = resources
        self.startupTimeout = startupTimeout
        self.shutdownGrace = shutdownGrace
    }

    func start() {
        guard state == .idle else { return }
        state = .starting
        launchID = UUID()
        nonce = UUID().uuidString.lowercased()
        stdoutBuffer.removeAll(keepingCapacity: true)

        let child = Process()
        let input = Pipe()
        let output = Pipe()
        let diagnostics = Pipe()
        child.executableURL = resources.host
        child.arguments = [
            "-root", resources.root.path,
            "-data", resources.data.path,
            "-assets", resources.web.path,
            "-port", "0",
            "-node", resources.node.path,
            "-bridge", resources.bridge.path,
            "-desktop-ready",
            "-desktop-nonce", nonce,
            "-parent-watch"
        ]
        child.environment = Self.childEnvironment()
        child.standardInput = input
        child.standardOutput = output
        child.standardError = diagnostics
        process = child
        stdinPipe = input
        stdoutPipe = output
        stderrPipe = diagnostics
        let generation = launchID

        child.terminationHandler = { [weak self] terminated in
            Task { @MainActor in self?.childTerminated(terminated, generation: generation) }
        }
        output.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            Task { @MainActor in self?.receive(data, generation: generation) }
        }
        // Drain diagnostics without retaining or displaying private paths or bodies.
        diagnostics.fileHandleForReading.readabilityHandler = { handle in _ = handle.availableData }

        do {
            try child.run()
        } catch {
            cleanupPipes()
            process = nil
            state = .idle
            onFailure?()
            return
        }

        let timeout = DispatchWorkItem { [weak self] in
            guard let self, self.launchID == generation, self.state == .starting else { return }
            self.failAndStop()
        }
        startupTimer = timeout
        DispatchQueue.main.asyncAfter(deadline: .now() + startupTimeout, execute: timeout)
    }

    func stop(completion: @escaping () -> Void) {
        stopCompletions.append(completion)
        guard let child = process else {
            finishStop()
            return
        }
        if state == .stopping { return }
        startupTimer?.cancel()
        startupTimer = nil
        state = .stopping
        // Closing the parent's only writer makes parent-watch mode observe EOF.
        stdinPipe?.fileHandleForWriting.closeFile()
        stdinPipe = nil
        let generation = launchID
        let timer = DispatchWorkItem { [weak self, weak child] in
            guard let self, let child, self.launchID == generation, self.state == .stopping, child.isRunning else { return }
            child.terminate()
            DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self, weak child] in
                guard let self, let child, self.launchID == generation, self.state == .stopping, child.isRunning else { return }
                // This PID belongs to the Process object created by this shell.
                _ = Darwin.kill(child.processIdentifier, SIGKILL)
            }
        }
        shutdownTimer = timer
        DispatchQueue.main.asyncAfter(deadline: .now() + shutdownGrace, execute: timer)
        if !child.isRunning { childTerminated(child, generation: generation) }
    }

    private static func childEnvironment() -> [String: String] {
        [
            "HOME": NSHomeDirectory(),
            "PATH": "/usr/bin:/bin:/usr/sbin:/sbin",
            "LANG": "en_US.UTF-8",
            "TMPDIR": FileManager.default.temporaryDirectory.path
        ]
    }

    private func receive(_ data: Data, generation: UUID) {
        guard launchID == generation else { return }
        guard state == .starting else {
            if !data.isEmpty { failAndStop() }
            return
        }
        if data.isEmpty { failAndStop(); return }
        stdoutBuffer.append(data)
        guard stdoutBuffer.count <= DesktopContract.readinessLimit else { failAndStop(); return }
        guard let newline = stdoutBuffer.firstIndex(of: 0x0A) else { return }
        let line = Data(stdoutBuffer[..<newline])
        let trailing = stdoutBuffer[stdoutBuffer.index(after: newline)...]
        guard trailing.allSatisfy({ $0 == 0x20 || $0 == 0x09 || $0 == 0x0D || $0 == 0x0A }) else { failAndStop(); return }
        guard let child = process,
              let ready = try? DesktopContract.decodeReadiness(line, expectedPID: child.processIdentifier, nonce: nonce)
        else { failAndStop(); return }

        startupTimer?.cancel()
        startupTimer = nil
        state = .ready
        onReady?(ready.origin)
    }

    private func failAndStop() {
        guard state == .starting || state == .ready else { return }
        onFailure?()
        stop(completion: {})
    }

    private func childTerminated(_ child: Process, generation: UUID) {
        guard launchID == generation, process === child else { return }
        let expected = state == .stopping
        startupTimer?.cancel()
        startupTimer = nil
        shutdownTimer?.cancel()
        shutdownTimer = nil
        child.terminationHandler = nil
        cleanupPipes()
        process = nil
        state = .idle
        if expected { finishStop() } else { onFailure?() }
    }

    private func cleanupPipes() {
        for pipe in [stdoutPipe, stderrPipe] {
            pipe?.fileHandleForReading.readabilityHandler = nil
            pipe?.fileHandleForReading.closeFile()
        }
        stdinPipe?.fileHandleForWriting.closeFile()
        stdoutPipe = nil
        stderrPipe = nil
        stdinPipe = nil
    }

    private func finishStop() {
        let completions = stopCompletions
        stopCompletions.removeAll()
        completions.forEach { $0() }
    }
}
