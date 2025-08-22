//
//  FloatingContentView.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import SwiftUI

struct FloatingContentView: View {
    let command: TopCommand?
    let queueInfo: QueueInfo?
    let onComplete: (() -> Void)?
    @State private var progress: Double = 1.0
    @State private var timeRemaining: TimeInterval = 0
    @State private var timer: Timer?
    @State private var isPaused: Bool = false
    
    init(command: TopCommand? = nil, queueInfo: QueueInfo? = nil, onComplete: (() -> Void)? = nil) {
        self.command = command
        self.queueInfo = queueInfo
        self.onComplete = onComplete
        if let command = command {
            self._timeRemaining = State(initialValue: command.durationInSeconds)
            self._progress = State(initialValue: 1.0)
        }
    }
    
    private var timeString: String {
        let minutes = Int(timeRemaining) / 60
        let seconds = Int(timeRemaining) % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
    
    var body: some View {
        if let command = command {
            ZStack(alignment: .leading) {
                Rectangle()
                    .fill(Color.black.opacity(0.8))
                    .frame(height: 50)
                
                Rectangle()
                    .fill(LinearGradient(
                        gradient: Gradient(colors: [.blue, .green]),
                        startPoint: .leading,
                        endPoint: .trailing
                    ))
                    .frame(height: 50)
                    .scaleEffect(x: progress, y: 1, anchor: .leading)
                    .animation(.linear(duration: 1), value: progress)
                
                HStack {
                    // Drag handle indicator
                    Image(systemName: "line.3.horizontal")
                        .foregroundColor(.white.opacity(0.6))
                        .font(.system(size: 12))
                        .padding(.leading, 8)
                    
                    VStack(alignment: .leading, spacing: 2) {
                        Text(command.text)
                            .font(.system(size: 14, weight: .medium))
                            .foregroundColor(.white)
                            .lineLimit(1)
                        
                        HStack(spacing: 4) {
                            Text(timeString)
                                .font(.system(size: 12))
                                .foregroundColor(.white.opacity(0.8))
                            
                            if isPaused {
                                Text("(Paused)")
                                    .font(.system(size: 10))
                                    .foregroundColor(.yellow.opacity(0.9))
                            }
                            
                            // Queue status indicator
                            if let queueInfo = queueInfo, queueInfo.totalCount > 1 {
                                Text("• \(queueInfo.currentPosition)/\(queueInfo.totalCount)")
                                    .font(.system(size: 10))
                                    .foregroundColor(.white.opacity(0.6))
                            }
                        }
                    }
                    .padding(.leading, 4)
                    
                    Spacer()
                    
                    // Control buttons
                    HStack(spacing: 8) {
                        // Pause/Resume button
                        Button(action: togglePause) {
                            Image(systemName: isPaused ? "play.circle.fill" : "pause.circle.fill")
                                .foregroundColor(.white)
                                .font(.system(size: 18))
                        }
                        .buttonStyle(PlainButtonStyle())
                        
                        // Dismiss button
                        Button(action: dismissFloatingBar) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.white.opacity(0.9))
                                .font(.system(size: 18))
                        }
                        .buttonStyle(PlainButtonStyle())
                    }
                    .padding(.trailing, 12)
                }
            }
            .cornerRadius(8)
            .shadow(color: .black.opacity(0.3), radius: 4, x: 0, y: 2)
            .frame(width: 400, height: 50)
            .onAppear {
                startTimer()
            }
            .onDisappear {
                stopTimer()
            }
        } else {
            Color.clear
                .frame(width: 1, height: 1)
        }
    }
    
    private func startTimer() {
        guard let command = command else { return }
        
        stopTimer()
        timeRemaining = command.durationInSeconds
        progress = 1.0
        isPaused = false
        
        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [self] _ in
            // Only update if not paused
            if !self.isPaused {
                self.timeRemaining = max(0, self.timeRemaining - 1)
                self.progress = command.durationInSeconds > 0 ? self.timeRemaining / command.durationInSeconds : 0
                
                if self.timeRemaining <= 0 {
                    self.completeCommand()
                }
            }
        }
    }
    
    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
    
    private func togglePause() {
        isPaused.toggle()
        print("DEBUG FloatingContentView: Timer \(isPaused ? "paused" : "resumed")")
    }
    
    private func completeCommand() {
        stopTimer()
        print("DEBUG FloatingContentView: Command completed, calling onComplete")
        onComplete?()
    }
    
    private func dismissFloatingBar() {
        stopTimer()
        print("DEBUG FloatingContentView: Dismiss button clicked, calling onComplete")
        onComplete?()
    }
}

#Preview {
    FloatingContentView(command: TopCommand(id: 1, text: "Sample Todo Task", duration: 30000000000))
}
