//
//  todo_stickerApp.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import SwiftUI
import SwiftData
import Cocoa

struct QueueInfo {
    let currentPosition: Int
    let totalCount: Int
    let remainingCount: Int
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
        }
        .modelContainer(sharedModelContainer)
    }
}

class AppDelegate: NSObject, NSApplicationDelegate, ObservableObject {
    private var activeFloatingWindows: [Int64: FloatingWindowController] = [:]
    private let windowHeight: CGFloat = 60
    private let windowSpacing: CGFloat = 10
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        startFloatingWindowMonitoring()
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
        
        // Create new floating window controller
        let windowController = FloatingWindowController()
        
        // Calculate position for this window
        let position = calculateWindowPosition(for: activeFloatingWindows.count)
        windowController.setPosition(position)
        
        // Create content view with completion callback
        let contentView = FloatingContentView(
            command: command,
            queueInfo: nil, // No queue info needed for simultaneous display
            onComplete: { [weak self] in
                self?.removeFloatingWindow(for: command.id)
            }
        )
        let hostingView = NSHostingView(rootView: contentView)
        windowController.window?.contentView = hostingView
        
        // Store and show the window
        activeFloatingWindows[command.id] = windowController
        windowController.showFloatingBar()
        
        print("DEBUG AppDelegate: Now showing \(activeFloatingWindows.count) floating windows")
    }
    
    private func removeFloatingWindow(for commandId: Int64) {
        print("DEBUG AppDelegate: Removing floating window for command ID: \(commandId)")
        
        if let windowController = activeFloatingWindows.removeValue(forKey: commandId) {
            windowController.hideFloatingBar()
        }
        
        // Reposition remaining windows
        repositionAllWindows()
        
        print("DEBUG AppDelegate: Now showing \(activeFloatingWindows.count) floating windows")
    }
    
    private func calculateWindowPosition(for index: Int) -> NSPoint {
        guard let screen = NSScreen.main else {
            return NSPoint(x: 100, y: 100)
        }
        
        let screenFrame = screen.visibleFrame
        let windowWidth: CGFloat = 400
        let x = screenFrame.midX - windowWidth / 2
        let y = screenFrame.maxY - 20 - CGFloat(index) * (windowHeight + windowSpacing)
        
        return NSPoint(x: x, y: y)
    }
    
    private func repositionAllWindows() {
        let sortedWindows = activeFloatingWindows.sorted { $0.key < $1.key }
        
        for (index, (_, windowController)) in sortedWindows.enumerated() {
            let newPosition = calculateWindowPosition(for: index)
            windowController.setPosition(newPosition)
        }
    }
    
    func hideAllFloatingBars() {
        print("DEBUG AppDelegate: Hiding all floating windows")
        
        for (_, windowController) in activeFloatingWindows {
            windowController.hideFloatingBar()
        }
        activeFloatingWindows.removeAll()
    }
}
