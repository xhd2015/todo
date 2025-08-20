//
//  CommandMonitor.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import Foundation
import Combine

struct TopCommand: Codable {
    let id: Int64
    let text: String
    let duration: Int64 // Duration in nanoseconds from Go
    
    // Convert nanoseconds to seconds
    var durationInSeconds: TimeInterval {
        return TimeInterval(duration) / 1_000_000_000.0
    }
}

class CommandMonitor: ObservableObject, CommandMonitorProtocol {
    @Published var currentCommand: TopCommand?
    @Published var progress: Double = 1.0
    @Published var timeRemaining: TimeInterval = 0
    @Published var receivedCommands: [TopCommand] = []
    
    private var timer: Timer?
    private var fileMonitor: DispatchSourceFileSystemObject?
    private var pollingTimer: Timer?
    private let commandFileURL: URL
    private var totalDuration: TimeInterval = 0
    private var lastFileModificationDate: Date?
    
    init() {
        let homeDir = FileManager.default.homeDirectoryForCurrentUser
        let commandDir = homeDir.appendingPathComponent(".todo-sticker")
        self.commandFileURL = commandDir.appendingPathComponent("command.json")
        
        // Create directory if it doesn't exist
        try? FileManager.default.createDirectory(at: commandDir, withIntermediateDirectories: true)
    }
    
    func startMonitoring() {
        print("DEBUG CommandMonitor: Starting file monitoring...")
        print("DEBUG CommandMonitor: Monitoring file at: \(commandFileURL.path)")
        
        guard fileMonitor == nil else { 
            print("DEBUG CommandMonitor: File monitor already exists, skipping")
            return 
        }
        
        // Create directory if it doesn't exist
        let commandDir = commandFileURL.deletingLastPathComponent()
        if !FileManager.default.fileExists(atPath: commandDir.path) {
            do {
                try FileManager.default.createDirectory(at: commandDir, withIntermediateDirectories: true)
                print("DEBUG CommandMonitor: Created directory: \(commandDir.path)")
            } catch {
                print("DEBUG CommandMonitor: Failed to create directory: \(error)")
                return
            }
        }
        
        // Create file if it doesn't exist
        if !FileManager.default.fileExists(atPath: commandFileURL.path) {
            let success = FileManager.default.createFile(atPath: commandFileURL.path, contents: nil)
            print("DEBUG CommandMonitor: Created file: \(success)")
        }
        
        let fileDescriptor = open(commandFileURL.path, O_EVTONLY)
        guard fileDescriptor >= 0 else {
            print("DEBUG CommandMonitor: Failed to open file for monitoring, descriptor: \(fileDescriptor)")
            return
        }
        
        print("DEBUG CommandMonitor: File descriptor opened successfully: \(fileDescriptor)")
        
        fileMonitor = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: fileDescriptor,
            eventMask: .write,
            queue: DispatchQueue.main
        )
        
        fileMonitor?.setEventHandler { [weak self] in
            print("DEBUG CommandMonitor: File change detected!")
            self?.handleFileChange()
        }
        
        fileMonitor?.setCancelHandler {
            print("DEBUG CommandMonitor: File monitor cancelled, closing descriptor")
            close(fileDescriptor)
        }
        
        fileMonitor?.resume()
        print("DEBUG CommandMonitor: File monitor started and resumed")
        
        // Also start polling as a backup mechanism
        startPolling()
        
        // Check for existing command on startup
        print("DEBUG CommandMonitor: Checking for existing command on startup")
        handleFileChange()
    }
    
    func stopMonitoring() {
        fileMonitor?.cancel()
        fileMonitor = nil
        stopPolling()
    }
    
    private func startPolling() {
        print("DEBUG CommandMonitor: Starting polling mechanism as backup")
        pollingTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.checkFileModification()
        }
    }
    
    private func stopPolling() {
        pollingTimer?.invalidate()
        pollingTimer = nil
    }
    
    private func checkFileModification() {
        guard FileManager.default.fileExists(atPath: commandFileURL.path) else { return }
        
        do {
            let attributes = try FileManager.default.attributesOfItem(atPath: commandFileURL.path)
            if let modificationDate = attributes[.modificationDate] as? Date {
                if lastFileModificationDate == nil || modificationDate > lastFileModificationDate! {
                    print("DEBUG CommandMonitor: File modification detected via polling")
                    lastFileModificationDate = modificationDate
                    handleFileChange()
                }
            }
        } catch {
            // Ignore errors in polling
        }
    }
    
    private func handleFileChange() {
        print("DEBUG CommandMonitor: handleFileChange called")
        
        // Check if file exists
        guard FileManager.default.fileExists(atPath: commandFileURL.path) else {
            print("DEBUG CommandMonitor: Command file does not exist")
            return
        }
        
        // Try to read file data
        guard let data = try? Data(contentsOf: commandFileURL) else {
            print("DEBUG CommandMonitor: Failed to read file data")
            return
        }
        
        print("DEBUG CommandMonitor: Read \(data.count) bytes from file")
        
        guard !data.isEmpty else { 
            print("DEBUG CommandMonitor: File is empty, skipping")
            return 
        }
        
        // Print raw data for debugging
        if let dataString = String(data: data, encoding: .utf8) {
            print("DEBUG CommandMonitor: File contents: \(dataString)")
        }
        
        do {
            let command = try JSONDecoder().decode(TopCommand.self, from: data)
            print("DEBUG CommandMonitor: Successfully decoded command - ID: \(command.id), Text: \(command.text), Duration: \(command.duration) ns (\(command.durationInSeconds) seconds)")
            
            DispatchQueue.main.async {
                print("DEBUG CommandMonitor: Setting currentCommand on main queue")
                
                // Add to received commands list
                self.receivedCommands.append(command)
                print("DEBUG CommandMonitor: Added command to list, total commands: \(self.receivedCommands.count)")
                
                // Set as current command for floating bar
                self.currentCommand = command
                self.totalDuration = command.durationInSeconds
                self.timeRemaining = command.durationInSeconds
                self.progress = 1.0
                print("DEBUG CommandMonitor: Command set, currentCommand is now: \(self.currentCommand != nil ? "not nil" : "nil")")
            }
            
            // Clear the file after reading
            do {
                try "".write(to: commandFileURL, atomically: true, encoding: .utf8)
                print("DEBUG CommandMonitor: File cleared successfully")
            } catch {
                print("DEBUG CommandMonitor: Failed to clear file: \(error)")
            }
            
        } catch {
            print("DEBUG CommandMonitor: Failed to decode command: \(error)")
        }
    }
    
    func startTimer() {
        stopTimer()
        
        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            guard let self = self else { return }
            
            self.timeRemaining = max(0, self.timeRemaining - 1)
            self.progress = self.totalDuration > 0 ? self.timeRemaining / self.totalDuration : 0
            
            if self.timeRemaining <= 0 {
                self.stopTimer()
            }
        }
    }
    
    func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
    
    func clearCommands() {
        print("DEBUG CommandMonitor: Clearing commands list")
        receivedCommands.removeAll()
    }
    
    deinit {
        stopTimer()
        stopMonitoring()
        stopPolling()
    }
}
