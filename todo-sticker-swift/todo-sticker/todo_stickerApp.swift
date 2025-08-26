//
//  todo_stickerApp.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import SwiftUI
import SwiftData
import Cocoa
import Combine

struct QueueInfo {
    let currentPosition: Int
    let totalCount: Int
    let remainingCount: Int
}

struct ServerInfo {
    let status: String
    let port: Int
    let receivedCommands: [TopCommand]
}

struct FloatingWindowInfo {
    let controller: FloatingWindowController
    let command: TopCommand
}

@main
struct todo_stickerApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    var sharedModelContainer: ModelContainer = {
        let schema = Schema([
            Item.self,
        ])
        let modelConfiguration = ModelConfiguration(schema: schema, isStoredInMemoryOnly: false)

        do {
            return try ModelContainer(for: schema, configurations: [modelConfiguration])
        } catch {
            fatalError("Could not create ModelContainer: \(error)")
        }
    }()

    var body: some Scene {
        WindowGroup {
            MainContentView()
                .environmentObject(appDelegate)
        }
        .modelContainer(sharedModelContainer)
    }
}

class AppDelegate: NSObject, NSApplicationDelegate, ObservableObject {
    private var floatingWindows: [UUID: FloatingWindowInfo] = [:]
    private let windowHeight: CGFloat = 60
    private let windowSpacing: CGFloat = 10
    
    // HTTP server instance (private, managed internally)
    private var commandMonitor: HTTPCommandMonitor
    
    // Published properties for SwiftUI to observe
    @Published var serverStatus: String = "Starting..."
    @Published var serverPort: Int = 4756
    @Published var receivedCommands: [TopCommand] = []
    
    // Combine cancellables for managing subscriptions
    private var cancellables = Set<AnyCancellable>()
    
    override init() {
        commandMonitor = HTTPCommandMonitor()
        super.init()
        setupServerCallbacks()
    }
    
    var serverInfo: ServerInfo {
        ServerInfo(
            status: serverStatus,
            port: serverPort,
            receivedCommands: receivedCommands
        )
    }
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        startHTTPServer()
        startFloatingWindowMonitoring()
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        stopHTTPServer()
    }
    
    private func startHTTPServer() {
        print("DEBUG AppDelegate: Starting HTTP server at application launch")
        commandMonitor.startMonitoring()
    }
    
    private func stopHTTPServer() {
        print("DEBUG AppDelegate: Stopping HTTP server at application termination")
        commandMonitor.stopMonitoring()
    }
    
    private func setupServerCallbacks() {
        // Observe server status changes and explicitly update SwiftUI @Published properties
        commandMonitor.objectWillChange.sink { [weak self] _ in
            DispatchQueue.main.async {
                // Explicitly update SwiftUI state when server state changes
                self?.serverStatus = self?.commandMonitor.serverStatus ?? "Unknown"
                self?.receivedCommands = self?.commandMonitor.receivedCommands ?? []
            }
        }
        .store(in: &cancellables)
    }
    
    // Public method for SwiftUI to call
    func clearReceivedCommands() {
        commandMonitor.clearCommands()
        // Explicitly update SwiftUI state
        DispatchQueue.main.async { [weak self] in
            self?.receivedCommands = []
        }
    }
    

    
    private func startFloatingWindowMonitoring() {
        // Listen for command updates from the main window's HTTP monitor
        NotificationCenter.default.addObserver(
            forName: NSNotification.Name("CommandReceived"),
            object: nil,
            queue: .main
        ) { [weak self] notification in
            if let command = notification.object as? TopCommand {
                self?.addFloatingWindow(for: command)
            }
        }
    }
    
    private func addFloatingWindow(for command: TopCommand) {
        print("DEBUG AppDelegate: Adding floating window for command - ID: \(command.id), Text: \(command.text)")
        
        // Generate unique window ID to handle duplicates
        let windowId = UUID()
        
        // Create new floating window controller
        let windowController = FloatingWindowController()
        
        // Calculate position based on existing windows and screen bounds
        let position = calculateOptimalWindowPosition()
        windowController.setPosition(position)
        
        // Create content view with completion callback
        let contentView = FloatingContentView(
            command: command,
            queueInfo: nil, // No queue info needed for simultaneous display
            onComplete: { [weak self] in
                self?.removeFloatingWindow(for: windowId)
            }
        )
        let hostingView = NSHostingView(rootView: contentView)
        windowController.window?.contentView = hostingView
        
        // Store window info and show the window
        let windowInfo = FloatingWindowInfo(
            controller: windowController,
            command: command
        )
        floatingWindows[windowId] = windowInfo
        windowController.showFloatingBar()
        
        print("DEBUG AppDelegate: Now showing \(floatingWindows.count) floating windows")
    }
    
    private func removeFloatingWindow(for windowId: UUID) {
        print("DEBUG AppDelegate: Removing floating window for window ID: \(windowId)")
        
        if let windowInfo = floatingWindows.removeValue(forKey: windowId) {
            windowInfo.controller.hideFloatingBar()
        }
        
        // Note: We intentionally do NOT reposition remaining windows to avoid visual disruption
        // Each window maintains its original position when others are dismissed
        
        print("DEBUG AppDelegate: Now showing \(floatingWindows.count) floating windows")
    }
    
    private func calculateOptimalWindowPosition() -> NSPoint {
        guard let screen = NSScreen.main else {
            return NSPoint(x: 100, y: 100)
        }
        
        let screenFrame = screen.visibleFrame
        let windowWidth: CGFloat = 400
        let x = screenFrame.midX - windowWidth / 2
        
        // Find the lowest Y position among existing windows
        var lowestY = screenFrame.maxY - 20 // Start from top with margin
        
        if !floatingWindows.isEmpty {
            // Get current positions of all windows (not stored positions)
            let currentPositions = floatingWindows.values.compactMap { windowInfo -> NSPoint? in
                return windowInfo.controller.window?.frame.origin
            }
            
            if !currentPositions.isEmpty {
                // Find the most bottom window based on current positions
                let bottomMostY = currentPositions.min { $0.y < $1.y }?.y ?? lowestY
                lowestY = bottomMostY - windowHeight - windowSpacing
            }
        }
        
        // Ensure the window stays within screen bounds
        let minY = screenFrame.minY + 20 // Bottom margin
        let maxY = screenFrame.maxY - windowHeight - 20 // Top margin
        
        // Adjust Y to ensure full visibility
        var finalY = lowestY
        if finalY < minY {
            // If it would go below screen, place it at the bottom with margin
            finalY = minY
        } else if finalY > maxY {
            // If it would go above screen, place it at the top with margin
            finalY = maxY
        }
        
        // Ensure X position keeps window fully visible
        let minX = screenFrame.minX + 10 // Left margin
        let maxX = screenFrame.maxX - windowWidth - 10 // Right margin
        let finalX = max(minX, min(maxX, x))
        
        return NSPoint(x: finalX, y: finalY)
    }
    

    
    func hideAllFloatingBars() {
        print("DEBUG AppDelegate: Hiding all floating windows")
        
        for (_, windowInfo) in floatingWindows {
            windowInfo.controller.hideFloatingBar()
        }
        floatingWindows.removeAll()
    }
}
