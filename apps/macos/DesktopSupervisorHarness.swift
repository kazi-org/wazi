import AppKit
import Darwin
import Foundation

@MainActor
enum DesktopSupervisorHarness {
    private enum Outcome: Equatable { case ready, failed }
    private enum Scenario: String, CaseIterable, Equatable {
        case wrongNonce, wrongProtocol, wrongOrigin, exited, timeout, validStop
    }

    static func run() throws {
        let base = FileManager.default.temporaryDirectory
            .appendingPathComponent("wazi-supervisor-test-\(UUID().uuidString)", isDirectory: true)
        let resourcesURL = base.appendingPathComponent("resources", isDirectory: true)
        let dataURL = base.appendingPathComponent("data", isDirectory: true)
        let rootURL = base.appendingPathComponent("root", isDirectory: true)
        let webURL = resourcesURL.appendingPathComponent("web", isDirectory: true)
        try FileManager.default.createDirectory(at: webURL, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: dataURL, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: rootURL, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: base) }

        for scenario in Scenario.allCases {
            let hostURL = resourcesURL.appendingPathComponent("mock-host", isDirectory: false)
            try writeMockHost(scenario: scenario, to: hostURL)
            let resources = DesktopResources(
                host: hostURL,
                node: hostURL,
                bridge: hostURL,
                root: rootURL,
                data: dataURL,
                web: webURL
            )
            let supervisor = HostSupervisor(resources: resources, startupTimeout: 0.6, shutdownGrace: 0.3)
            var outcome: Outcome?
            supervisor.onReady = { _ in
                if outcome == nil { outcome = .ready }
            }
            supervisor.onFailure = {
                if outcome == nil { outcome = .failed }
            }
            supervisor.start()

            guard pump(until: { outcome != nil }, seconds: 3) else {
                supervisor.stop(completion: {})
                throw HarnessError.timedOut(scenario.rawValue)
            }
            let expected: Outcome = scenario == .validStop ? .ready : .failed
            guard outcome == expected else {
                supervisor.stop(completion: {})
                guard pump(until: { !supervisor.hasOwnedProcess }, seconds: 3) else {
                    throw HarnessError.childLeaked(scenario.rawValue)
                }
                throw HarnessError.wrongOutcome(scenario.rawValue)
            }

            if scenario == .validStop {
                var stopped = false
                supervisor.stop { stopped = true }
                guard pump(until: { stopped }, seconds: 3),
                      FileManager.default.fileExists(atPath: dataURL.appendingPathComponent("eof").path),
                      !supervisor.hasOwnedProcess else {
                    throw HarnessError.stopFailed
                }
            } else {
                guard pump(until: { !supervisor.hasOwnedProcess }, seconds: 3) else {
                    supervisor.stop(completion: {})
                    throw HarnessError.childLeaked(scenario.rawValue)
                }
            }
        }
    }

    private static func pump(until predicate: () -> Bool, seconds: TimeInterval) -> Bool {
        let deadline = Date().addingTimeInterval(seconds)
        while !predicate(), Date() < deadline {
            _ = RunLoop.main.run(mode: .default, before: min(deadline, Date().addingTimeInterval(0.02)))
        }
        return predicate()
    }

    private static func writeMockHost(scenario: Scenario, to url: URL) throws {
        let mode = scenario.rawValue
        let script = #"""
#!/bin/sh
nonce=
data=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -desktop-nonce) nonce="$2"; shift 2 ;;
    -data) data="$2"; shift 2 ;;
    *) shift ;;
  esac
done
mode='MODE'
if [ "$mode" = exited ]; then exit 0; fi
if [ "$mode" = timeout ]; then IFS= read -r ignored || exit 0; fi
protocol='wazi-desktop/1'
origin='http://127.0.0.1:38123'
case "$mode" in
  wrongNonce) nonce=bad-nonce ;;
  wrongProtocol) protocol=wrong-protocol ;;
  wrongOrigin) origin='http://localhost:38123' ;;
esac
if [ "$mode" = wrongNonce ] || [ "$mode" = wrongProtocol ] || [ "$mode" = wrongOrigin ] || [ "$mode" = validStop ]; then
  printf '{"protocol":"%s","version":"0.1.0","origin":"%s","pid":%s,"nonce":"%s"}\n' "$protocol" "$origin" "$$" "$nonce"
fi
if [ "$mode" = validStop ]; then
  while IFS= read -r ignored; do :; done
  printf eof > "$data/eof"
fi
"""#.replacingOccurrences(of: "MODE", with: mode)
        try script.write(to: url, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o700], ofItemAtPath: url.path)
    }

    enum HarnessError: Error {
        case timedOut(String)
        case wrongOutcome(String)
        case childLeaked(String)
        case stopFailed
    }
}
