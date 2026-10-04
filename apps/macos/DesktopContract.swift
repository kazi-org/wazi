import CoreFoundation
import Foundation

enum DesktopContract {
    static let protocolName = "wazi-desktop/1"
    static let hostVersion = "0.1.0"
    static let readinessLimit = 4_096
    static let startupTimeout: TimeInterval = 15
    static let shutdownGrace: TimeInterval = 5
    static let originHost = "127.0.0.1"

    enum Action: String, CaseIterable {
        case close
        case minimize
        case fullscreen
        case drag
    }

    struct Ready {
        let origin: URL
    }

    enum ValidationError: Error {
        case oversize
        case malformed
        case mismatchedProcess
        case mismatchedNonce
        case invalidOrigin
    }

    static func decodeReadiness(_ data: Data, expectedPID: Int32, nonce: String) throws -> Ready {
        guard data.count <= readinessLimit else { throw ValidationError.oversize }
        guard let object = try? JSONSerialization.jsonObject(with: data),
              let fields = object as? [String: Any],
              Set(fields.keys) == Set(["protocol", "version", "origin", "pid", "nonce"]),
              fields["protocol"] as? String == protocolName,
              fields["version"] as? String == hostVersion,
              let originString = fields["origin"] as? String,
              let processNumber = fields["pid"] as? NSNumber,
              CFGetTypeID(processNumber) != CFBooleanGetTypeID(),
              processNumber.doubleValue.isFinite,
              processNumber.doubleValue.rounded(.towardZero) == processNumber.doubleValue
        else { throw ValidationError.malformed }

        guard expectedPID > 0,
              processNumber.doubleValue == Double(expectedPID) else { throw ValidationError.mismatchedProcess }
        guard fields["nonce"] as? String == nonce else { throw ValidationError.mismatchedNonce }
        guard let origin = validatedLoopbackOrigin(originString) else { throw ValidationError.invalidOrigin }
        return Ready(origin: origin)
    }

    static func validatedLoopbackOrigin(_ value: String) -> URL? {
        guard let components = URLComponents(string: value),
              components.scheme == "http",
              components.host == originHost,
              let port = components.port,
              (1...65_535).contains(port),
              components.user == nil,
              components.password == nil,
              components.percentEncodedPath.isEmpty,
              components.query == nil,
              components.fragment == nil,
              value == "http://\(originHost):\(port)"
        else { return nil }
        return URL(string: value)
    }

    static func matches(_ origin: URL, scheme: String, host: String, port: Int) -> Bool {
        guard let expected = URLComponents(url: origin, resolvingAgainstBaseURL: false),
              expected.scheme == "http",
              expected.host == originHost,
              let expectedPort = expected.port
        else { return false }
        return scheme == expected.scheme && host == expected.host && port == expectedPort
    }

    static func validatedExternalHTTPURL(_ url: URL) -> URL? {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              scheme == "http" || scheme == "https",
              let host = components.host, !host.isEmpty,
              components.user == nil,
              components.password == nil,
              components.port.map({ (1...65_535).contains($0) }) ?? true
        else { return nil }
        return components.url
    }

    static func action(from body: Any) -> Action? {
        guard let fields = body as? [String: Any],
              Set(fields.keys) == Set(["action"]),
              let raw = fields["action"] as? String
        else { return nil }
        return Action(rawValue: raw)
    }

    // The web drag rectangle is frozen by the coordinator-owned CSS contract:
    // x=112...224, y=4...30 from the top of the 34px header at page zoom 1.
    static func isReservedDragPoint(x: Double, y: Double, viewHeight: Double, isFlipped: Bool) -> Bool {
        guard x.isFinite, y.isFinite, viewHeight.isFinite, viewHeight >= 30 else { return false }
        let cssY = isFlipped ? y : viewHeight - y
        return x >= 112 && x < 224 && cssY >= 4 && cssY < 30
    }

    static func selfTest() throws {
        let nonce = "00000000-0000-4000-8000-000000000001"
        let pid: Int32 = 4321
        let validJSON = #"{"protocol":"wazi-desktop/1","version":"0.1.0","origin":"http://127.0.0.1:42123","pid":4321,"nonce":"00000000-0000-4000-8000-000000000001"}"#
        _ = try decodeReadiness(Data(validJSON.utf8), expectedPID: pid, nonce: nonce)

        let badPID = validJSON.replacingOccurrences(of: "\"pid\":4321", with: "\"pid\":4322")
        try expectFailure { _ = try decodeReadiness(Data(badPID.utf8), expectedPID: pid, nonce: nonce) }
        let badNonce = validJSON.replacingOccurrences(of: nonce, with: "00000000-0000-4000-8000-000000000002")
        try expectFailure { _ = try decodeReadiness(Data(badNonce.utf8), expectedPID: pid, nonce: nonce) }
        try expectFailure { _ = try decodeReadiness(Data(repeating: 0x20, count: readinessLimit + 1), expectedPID: pid, nonce: nonce) }

        for invalid in [
            "https://127.0.0.1:42123", "http://localhost:42123", "http://127.0.0.2:42123",
            "http://user@127.0.0.1:42123", "http://127.0.0.1:0", "http://127.0.0.1:65536",
            "http://127.0.0.1:42123/path", "http://127.0.0.1:42123?query=1", "http://127.0.0.1:42123#fragment"
        ] {
            guard validatedLoopbackOrigin(invalid) == nil else { throw SelfTestFailure("accepted invalid origin: \(invalid)") }
        }
        guard validatedLoopbackOrigin("http://127.0.0.1:42123") != nil else { throw SelfTestFailure("rejected valid origin") }

        guard action(from: ["action": "close"]) == .close,
              action(from: ["action": "drag"]) == .drag,
              action(from: ["action": "launch-shell"]) == nil,
              action(from: ["action": "close", "path": "/tmp/file"]) == nil,
              action(from: "close") == nil
        else { throw SelfTestFailure("window action validation failed") }

        guard isReservedDragPoint(x: 112, y: 700 - 10, viewHeight: 700, isFlipped: false),
              isReservedDragPoint(x: 223.9, y: 29.9, viewHeight: 700, isFlipped: true),
              !isReservedDragPoint(x: 224, y: 20, viewHeight: 700, isFlipped: true),
              !isReservedDragPoint(x: 111.9, y: 20, viewHeight: 700, isFlipped: true)
        else { throw SelfTestFailure("drag hit-region validation failed") }

        guard validatedExternalHTTPURL(URL(string: "https://example.com/path")!) != nil,
              validatedExternalHTTPURL(URL(string: "file:///etc/passwd")!) == nil,
              validatedExternalHTTPURL(URL(string: "javascript:alert(1)")!) == nil,
              validatedExternalHTTPURL(URL(string: "https://user@example.com/")!) == nil
        else { throw SelfTestFailure("external URL validation failed") }
    }

    private static func expectFailure(_ body: () throws -> Void) throws {
        do { try body() } catch { return }
        throw SelfTestFailure("invalid input was accepted")
    }
}

struct SelfTestFailure: Error, CustomStringConvertible {
    let message: String
    init(_ message: String) { self.message = message }
    var description: String { message }
}
