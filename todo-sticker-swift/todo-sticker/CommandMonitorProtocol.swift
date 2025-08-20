//
//  CommandMonitorProtocol.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import Foundation
import Combine

protocol CommandMonitorProtocol: ObservableObject {
    var currentCommand: TopCommand? { get }
    var progress: Double { get }
    var timeRemaining: TimeInterval { get }
    var receivedCommands: [TopCommand] { get }
    
    func startMonitoring()
    func stopMonitoring()
    func startTimer()
    func stopTimer()
    func clearCommands()
}
