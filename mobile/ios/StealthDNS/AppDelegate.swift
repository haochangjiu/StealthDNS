import UIKit

@main
class AppDelegate: UIResponder, UIApplicationDelegate {

    var window: UIWindow?
    var nhpInitialized = false

    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        
        // Initialize NHP Core in background
        initializeNHPCoreAsync()
        
        // Create window programmatically
        window = UIWindow(frame: UIScreen.main.bounds)
        window?.rootViewController = BrowserViewController()
        window?.makeKeyAndVisible()
        
        return true
    }
    
    private func initializeNHPCoreAsync() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            // Get documents directory for NHP working directory
            let documentsPath = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0].path
            
            // Create necessary directories
            let etcDir = (documentsPath as NSString).appendingPathComponent("etc")
            let logsDir = (documentsPath as NSString).appendingPathComponent("logs")
            try? FileManager.default.createDirectory(atPath: etcDir, withIntermediateDirectories: true)
            try? FileManager.default.createDirectory(atPath: logsDir, withIntermediateDirectories: true)
            
            // Copy config files
            self?.copyConfigFiles(to: etcDir)
            
            // Load config JSON and initialize
            guard let configJSON = self?.loadConfigJSON() else {
                print("Failed to load config JSON")
                return
            }
            
            do {
                try NhpcoreInitializeWithConfig(documentsPath, 4, configJSON)
                self?.nhpInitialized = true
                print("NHP Core initialized successfully")
            } catch {
                print("Failed to initialize NHP Core: \(error)")
            }
        }
    }
    
    private func loadConfigJSON() -> String? {
        guard let resourcesURL = Bundle.main.url(forResource: "resources", withExtension: "json"),
              let data = try? Data(contentsOf: resourcesURL),
              let jsonString = String(data: data, encoding: .utf8) else {
            return nil
        }
        return jsonString
    }
    
    private func copyConfigFiles(to etcDir: String) {
        guard let resourcesURL = Bundle.main.url(forResource: "resources", withExtension: "json"),
              let data = try? Data(contentsOf: resourcesURL),
              let config = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return
        }
        
        // Create config.toml
        if let agent = config["agent"] as? [String: Any] {
            var configToml = ""
            configToml += "PrivateKeyBase64 = \"\(agent["privateKeyBase64"] as? String ?? "")\"\n"
            configToml += "DefaultCipherScheme = \(agent["cipherScheme"] as? Int ?? 1)\n"
            configToml += "UserId = \"\(agent["userId"] as? String ?? "mobile-user")\"\n"
            configToml += "OrganizationId = \"\(agent["organizationId"] as? String ?? "")\"\n"
            configToml += "LogLevel = 4\n"
            
            let configPath = (etcDir as NSString).appendingPathComponent("config.toml")
            try? configToml.write(toFile: configPath, atomically: true, encoding: .utf8)
        }
        
        // Create server.toml
        if let servers = config["servers"] as? [[String: Any]] {
            var serverToml = ""
            for srv in servers {
                serverToml += "[[Servers]]\n"
                serverToml += "Hostname = \"\(srv["hostname"] as? String ?? "")\"\n"
                serverToml += "Ip = \"\(srv["ip"] as? String ?? "")\"\n"
                serverToml += "Port = \(srv["port"] as? Int ?? 62206)\n"
                serverToml += "PubKeyBase64 = \"\(srv["pubKeyBase64"] as? String ?? "")\"\n"
                serverToml += "ExpireTime = \(srv["expireTime"] as? Int ?? 0)\n\n"
            }
            
            let serverPath = (etcDir as NSString).appendingPathComponent("server.toml")
            try? serverToml.write(toFile: serverPath, atomically: true, encoding: .utf8)
        }
        
        // Create resource.toml
        if let resources = config["resources"] as? [[String: Any]] {
            var resourceToml = ""
            for res in resources {
                resourceToml += "[[Resources]]\n"
                resourceToml += "AuthServiceId = \"\(res["authServiceId"] as? String ?? "")\"\n"
                resourceToml += "ResourceId = \"\(res["resourceId"] as? String ?? "")\"\n"
                resourceToml += "ServerHostname = \"\(res["serverHostname"] as? String ?? "")\"\n"
                resourceToml += "ServerIp = \"\(res["serverIp"] as? String ?? "")\"\n"
                resourceToml += "ServerPort = \(res["serverPort"] as? Int ?? 62206)\n\n"
            }
            
            let resourcePath = (etcDir as NSString).appendingPathComponent("resource.toml")
            try? resourceToml.write(toFile: resourcePath, atomically: true, encoding: .utf8)
        }
        
        print("Config files created in \(etcDir)")
    }
    
    func application(_ app: UIApplication, open url: URL, options: [UIApplication.OpenURLOptionsKey : Any] = [:]) -> Bool {
        // Handle URL schemes (nhp://, stealthdns://)
        if url.scheme == "nhp" || url.scheme == "stealthdns" {
            if let browserVC = window?.rootViewController as? BrowserViewController {
                browserVC.loadURL(url.absoluteString)
            }
            return true
        }
        return false
    }
    
    func applicationWillTerminate(_ application: UIApplication) {
        // Cleanup NHP resources
        if nhpInitialized {
            NhpcoreCleanup()
        }
    }
}
