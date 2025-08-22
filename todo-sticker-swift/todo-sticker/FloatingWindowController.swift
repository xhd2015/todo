//
//  FloatingWindowController.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import Cocoa
import SwiftUI

class FloatingWindow: NSWindow {
    override var canBecomeKey: Bool { false }
    override var canBecomeMain: Bool { false }
    override var acceptsFirstResponder: Bool { false }
}

class FloatingWindowController: NSWindowController {
    
    convenience init() {
        let window = FloatingWindow(
            contentRect: NSRect(x: 0, y: 0, width: 400, height: 60),
            styleMask: [.borderless],
            backing: .buffered,
            defer: false
        )
        
        self.init(window: window)
        configureWindow()
    }
    
    private func configureWindow() {
        guard let window = window else { return }
        
        // Make window float above all other applications
        window.level = NSWindow.Level.floating
        
        // Make window transparent
        window.backgroundColor = NSColor.clear
        window.isOpaque = false
        window.hasShadow = true
        
        // Remove window decorations
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .hidden
        
        // Make window non-resizable but movable
        window.styleMask.remove(.resizable)
        window.isMovable = true
        window.isMovableByWindowBackground = true
        
        // Enable mouse events
        window.acceptsMouseMovedEvents = true
        
        // Position window at top center of screen
        if let screen = NSScreen.main {
            let screenFrame = screen.visibleFrame
            let windowWidth: CGFloat = 400
            let windowHeight: CGFloat = 60
            let x = screenFrame.midX - windowWidth / 2
            let y = screenFrame.maxY - windowHeight - 20 // 20px from top
            
            window.setFrame(NSRect(x: x, y: y, width: windowWidth, height: windowHeight), display: true)
        }
        
        // Make window appear in all spaces and prevent activation
        window.collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle]
    }
    
    func setPosition(_ position: NSPoint) {
        guard let window = window else { return }
        let windowSize = window.frame.size
        window.setFrame(NSRect(origin: position, size: windowSize), display: true)
    }
    
    func showFloatingBar() {
        // Show window without making it key to prevent focus switching
        window?.orderFrontRegardless()
    }
    
    func hideFloatingBar() {
        window?.orderOut(nil)
    }
}
