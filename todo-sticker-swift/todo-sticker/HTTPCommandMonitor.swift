//
//  HTTPCommandMonitor.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import Foundation
import Combine
import Network

class HTTPCommandMonitor: ObservableObject, CommandMonitorProtocol {
    @Published var currentCommand: TopCommand?
    @Published var progress: Double = 1.0
    @Published var timeRemaining: TimeInterval = 0
    @Published var receivedCommands: [TopCommand] = []
    @Published var serverStatus: String = "Starting..."
    
    private var timer: Timer?
    private var httpServer: HTTPServer?
    private var totalDuration: TimeInterval = 0
    
    init() {
        httpServer = HTTPServer()
    }
    
    func startMonitoring() {
        print("DEBUG HTTPCommandMonitor: Starting HTTP server monitoring...")
        
        httpServer?.onCommandReceived = { [weak self] command in
            print("DEBUG HTTPCommandMonitor: Command received via HTTP - ID: \(command.id), Text: \(command.text), Duration: \(command.durationInSeconds) seconds")
            
            DispatchQueue.main.async {
                print("DEBUG HTTPCommandMonitor: Setting currentCommand on main queue")
                
                // Add to received commands list
                self?.receivedCommands.append(command)
                print("DEBUG HTTPCommandMonitor: Added command to list, total commands: \(self?.receivedCommands.count ?? 0)")
                
                // Set as current command for floating bar
                self?.currentCommand = command
                self?.totalDuration = command.durationInSeconds
                self?.timeRemaining = command.durationInSeconds
                self?.progress = 1.0
                print("DEBUG HTTPCommandMonitor: Command set, currentCommand is now: \(self?.currentCommand != nil ? "not nil" : "nil")")
                
                // Send notification for floating window
                NotificationCenter.default.post(
                    name: NSNotification.Name("CommandReceived"),
                    object: command
                )
            }
        }
        
        httpServer?.onStatusUpdate = { [weak self] status in
            DispatchQueue.main.async {
                self?.serverStatus = status
            }
        }
        
        httpServer?.start(port: 4756)
    }
    
    func stopMonitoring() {
        print("DEBUG HTTPCommandMonitor: Stopping HTTP server")
        httpServer?.stop()
    }
    
    func startTimer() {
        stopTimer()
        print("DEBUG HTTPCommandMonitor: Starting countdown timer")
        
        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            guard let self = self else { return }
            
            self.timeRemaining = max(0, self.timeRemaining - 1)
            self.progress = self.totalDuration > 0 ? self.timeRemaining / self.totalDuration : 0
            
            if self.timeRemaining <= 0 {
                print("DEBUG HTTPCommandMonitor: Timer finished")
                self.stopTimer()
            }
        }
    }
    
    func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
    
    func clearCommands() {
        print("DEBUG HTTPCommandMonitor: Clearing commands list")
        receivedCommands.removeAll()
    }
    
    deinit {
        stopTimer()
        stopMonitoring()
    }
}

class HTTPServer {
    private var listener: NWListener?
    var onCommandReceived: ((TopCommand) -> Void)?
    var onStatusUpdate: ((String) -> Void)?
    
    func start(port: UInt16) {
        print("DEBUG HTTPServer: Starting server on port \(port)")
        
        let parameters = NWParameters.tcp
        parameters.allowLocalEndpointReuse = true
        parameters.requiredLocalEndpoint = NWEndpoint.hostPort(host: "127.0.0.1", port: NWEndpoint.Port(rawValue: port)!)
        
        do {
            listener = try NWListener(using: parameters)
        } catch {
            print("DEBUG HTTPServer: Failed to create listener on port \(port): \(error)")
            let errorMessage = "Failed to create server on port \(port): \(error.localizedDescription)"
            
            // Only try alternative ports if the error indicates port is in use
            if isPortInUseError(error) {
                print("DEBUG HTTPServer: Port \(port) is in use, trying alternative ports")
                onStatusUpdate?("Port \(port) in use, trying alternatives...")
                tryAlternativePorts(startingFrom: port + 1)
            } else {
                print("DEBUG HTTPServer: Non-port-conflict error, not trying alternatives")
                onStatusUpdate?(errorMessage)
            }
            return
        }
        
        listener?.newConnectionHandler = { [weak self] connection in
            self?.handleConnection(connection)
        }
        
        listener?.stateUpdateHandler = { [weak self] state in
            print("DEBUG HTTPServer: Listener state: \(state)")
            switch state {
            case .ready:
                print("DEBUG HTTPServer: Server is ready and listening")
                self?.onStatusUpdate?("Server running on port \(port)")
            case .failed(let error):
                print("DEBUG HTTPServer: Server failed: \(error)")
                let errorMessage = "Server failed on port \(port): \(error.localizedDescription)"
                
                // Only try alternative ports if the error indicates port is in use
                if self?.isPortInUseError(error) == true {
                    print("DEBUG HTTPServer: Port \(port) is in use, trying alternative ports")
                    self?.onStatusUpdate?("Port \(port) in use, trying alternatives...")
                    self?.tryAlternativePorts(startingFrom: port + 1)
                } else {
                    print("DEBUG HTTPServer: Non-port-conflict error, not trying alternatives")
                    self?.onStatusUpdate?(errorMessage)
                }
            case .cancelled:
                print("DEBUG HTTPServer: Server was cancelled")
                self?.onStatusUpdate?("Server stopped")
            default:
                break
            }
        }
        
        listener?.start(queue: .global())
        print("DEBUG HTTPServer: Server start initiated")
    }
    
    private func tryAlternativePorts(startingFrom: UInt16) {
        let maxAttempts = 10
        for portOffset in 0..<maxAttempts {
            let alternativePort = startingFrom + UInt16(portOffset)
            print("DEBUG HTTPServer: Trying alternative port \(alternativePort)")
            
            let parameters = NWParameters.tcp
            parameters.allowLocalEndpointReuse = true
            parameters.requiredLocalEndpoint = NWEndpoint.hostPort(host: "127.0.0.1", port: NWEndpoint.Port(rawValue: alternativePort)!)
            
            do {
                let alternativeListener = try NWListener(using: parameters)
                
                alternativeListener.newConnectionHandler = { [weak self] connection in
                    self?.handleConnection(connection)
                }
                
                alternativeListener.stateUpdateHandler = { [weak self] state in
                    print("DEBUG HTTPServer: Alternative listener on port \(alternativePort) state: \(state)")
                    switch state {
                    case .ready:
                        print("DEBUG HTTPServer: Successfully started on alternative port \(alternativePort)")
                        self?.onStatusUpdate?("Server running on port \(alternativePort)")
                    case .failed(let error):
                        print("DEBUG HTTPServer: Alternative port \(alternativePort) failed: \(error)")
                        // If this alternative port also fails with a non-port-conflict error, stop trying
                        if self?.isPortInUseError(error) == false {
                            print("DEBUG HTTPServer: Alternative port failed with non-port-conflict error, stopping attempts")
                            self?.onStatusUpdate?("Server failed: \(error.localizedDescription)")
                        }
                    default:
                        break
                    }
                }
                
                alternativeListener.start(queue: .global())
                self.listener = alternativeListener
                print("DEBUG HTTPServer: Started on alternative port \(alternativePort)")
                return
            } catch {
                print("DEBUG HTTPServer: Alternative port \(alternativePort) failed: \(error)")
                
                // If this is not a port-in-use error, stop trying alternatives
                if !isPortInUseError(error) {
                    print("DEBUG HTTPServer: Alternative port creation failed with non-port-conflict error, stopping attempts")
                    onStatusUpdate?("Server failed: \(error.localizedDescription)")
                    return
                }
                continue
            }
        }
        
        print("DEBUG HTTPServer: Failed to start on any port from \(startingFrom) to \(startingFrom + UInt16(maxAttempts - 1))")
    }
    
    func stop() {
        listener?.cancel()
        listener = nil
    }
    
    private func isPortInUseError(_ error: Error) -> Bool {
        // Check if the error indicates the port is already in use
        let errorString = error.localizedDescription.lowercased()
        
        // Common port-in-use error indicators
        let portInUseIndicators = [
            "address already in use",
            "bind: address already in use",
            "port already in use",
            "eaddrinuse"
        ]
        
        for indicator in portInUseIndicators {
            if errorString.contains(indicator) {
                return true
            }
        }
        
        // Check for POSIX error codes that indicate port conflicts
        if let posixError = error as? POSIXError {
            return posixError.code == .EADDRINUSE
        }
        
        return false
    }
    
    private func handleConnection(_ connection: NWConnection) {
        print("DEBUG HTTPServer: New connection received")
        
        connection.start(queue: .global())
        
        connection.receive(minimumIncompleteLength: 1, maximumLength: 65536) { [weak self] data, _, isComplete, error in
            if let error = error {
                print("DEBUG HTTPServer: Connection error: \(error)")
                return
            }
            
            if let data = data, !data.isEmpty {
                self?.handleHTTPRequest(data: data, connection: connection)
            }
            
            if isComplete {
                connection.cancel()
            }
        }
    }
    
    private func handleHTTPRequest(data: Data, connection: NWConnection) {
        guard let requestString = String(data: data, encoding: .utf8) else {
            print("DEBUG HTTPServer: Failed to decode request data")
            return
        }
        
        print("DEBUG HTTPServer: Received request: \(requestString.prefix(200))")
        
        // Parse HTTP request
        let lines = requestString.components(separatedBy: "\r\n")
        guard let firstLine = lines.first else {
            sendResponse(connection: connection, status: "400 Bad Request", body: "Invalid request")
            return
        }
        
        let components = firstLine.components(separatedBy: " ")
        guard components.count >= 3,
              components[0] == "POST",
              components[1] == "/command" else {
            sendResponse(connection: connection, status: "404 Not Found", body: "Not found")
            return
        }
        
        // Find the JSON body
        if let bodyStartIndex = requestString.range(of: "\r\n\r\n")?.upperBound {
            let bodyString = String(requestString[bodyStartIndex...])
            if let bodyData = bodyString.data(using: .utf8) {
                do {
                    let command = try JSONDecoder().decode(TopCommand.self, from: bodyData)
                    print("DEBUG HTTPServer: Successfully decoded command: \(command)")
                    onCommandReceived?(command)
                    sendResponse(connection: connection, status: "200 OK", body: "Command received")
                } catch {
                    print("DEBUG HTTPServer: Failed to decode JSON: \(error)")
                    sendResponse(connection: connection, status: "400 Bad Request", body: "Invalid JSON")
                }
            }
        } else {
            sendResponse(connection: connection, status: "400 Bad Request", body: "No body found")
        }
    }
    
    private func sendResponse(connection: NWConnection, status: String, body: String) {
        let response = """
            HTTP/1.1 \(status)
            Content-Type: text/plain
            Content-Length: \(body.utf8.count)
            Connection: close
            
            \(body)
            """
        
        if let responseData = response.data(using: .utf8) {
            connection.send(content: responseData, completion: .contentProcessed { error in
                if let error = error {
                    print("DEBUG HTTPServer: Failed to send response: \(error)")
                }
                connection.cancel()
            })
        }
    }
}
