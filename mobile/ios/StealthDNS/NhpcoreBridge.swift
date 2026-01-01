import Foundation
import Nhpcore  // Import the Go library

// MARK: - NHP Core Bridge
// This file bridges the Go nhpcore library to Swift

/// Initialize the NHP core library with JSON config
func NhpcoreInitializeWithConfig(_ workDir: String, _ logLevel: Int, _ configJSON: String) throws {
    var error: NSError?
    NhpcoreInitializeWithConfig(workDir, Int(logLevel), configJSON, &error)
    if let error = error {
        throw error
    }
}

/// Check if a domain is an NHP domain
func NhpcoreIsNHPDomain(_ domain: String) -> Bool {
    return Nhpcore.NhpcoreIsNHPDomain(domain)
}

/// Extract resource ID from domain
func NhpcoreExtractResourceID(_ domain: String) -> String {
    return Nhpcore.NhpcoreExtractResourceID(domain)
}

/// Perform NHP knock and get result as JSON
func NhpcoreGetKnockResultJSON(_ resourceId: String) -> String {
    return Nhpcore.NhpcoreGetKnockResultJSON(resourceId)
}

/// Check if NHP is initialized
func NhpcoreIsInitialized() -> Bool {
    return Nhpcore.NhpcoreIsInitialized()
}

/// Cleanup NHP resources
func NhpcoreCleanup() {
    Nhpcore.NhpcoreCleanup()
}
