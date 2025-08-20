//
//  MainContentView.swift
//  todo-sticker
//
//  Created by xhd2015 on 8/20/25.
//

import SwiftUI
import SwiftData

struct MainContentView: View {
    @Environment(\.modelContext) private var modelContext
    @Query private var items: [Item]
    @StateObject private var commandMonitor = HTTPCommandMonitor()
    
    var body: some View {
        NavigationSplitView {
            VStack {
                // Server Status Section
                HStack {
                    Text("Server Status:")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    
                    Text(commandMonitor.serverStatus)
                        .font(.caption)
                        .foregroundColor(commandMonitor.serverStatus.contains("running") ? .green : .red)
                }
                .padding(.horizontal)
                
                Divider()
                
                // Received Commands Section
                if !commandMonitor.receivedCommands.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Received Commands")
                                .font(.headline)
                            
                            Spacer()
                            
                            Button("Clear") {
                                commandMonitor.clearCommands()
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        }
                        .padding(.horizontal)
                        
                        List(commandMonitor.receivedCommands, id: \.id) { command in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(command.text)
                                    .font(.body)
                                    .lineLimit(2)
                                
                                HStack {
                                    Text("ID: \(command.id)")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    
                                    Spacer()
                                    
                                    Text("Duration: \(Int(command.durationInSeconds / 60))m")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                            }
                            .padding(.vertical, 2)
                        }
                        .frame(maxHeight: 200)
                    }
                    
                    Divider()
                }
                
                // Original Items Section
                List {
                    ForEach(items) { item in
                        NavigationLink {
                            Text("Item at \(item.timestamp, format: Date.FormatStyle(date: .numeric, time: .standard))")
                        } label: {
                            Text(item.timestamp, format: Date.FormatStyle(date: .numeric, time: .standard))
                        }
                    }
                    .onDelete(perform: deleteItems)
                }
                .navigationTitle("Todo Sticker")
            }
            .navigationSplitViewColumnWidth(min: 180, ideal: 200)
            .toolbar {
                ToolbarItem {
                    Button(action: addItem) {
                        Label("Add Item", systemImage: "plus")
                    }
                }
            }
        } detail: {
            Text("Select an item")
        }
        .onAppear {
            print("DEBUG MainContentView: Starting command monitoring for main window")
            commandMonitor.startMonitoring()
        }
    }
    
    private func addItem() {
        withAnimation {
            let newItem = Item(timestamp: Date())
            modelContext.insert(newItem)
        }
    }
    
    private func deleteItems(offsets: IndexSet) {
        withAnimation {
            for index in offsets {
                modelContext.delete(items[index])
            }
        }
    }
}

#Preview {
    MainContentView()
        .modelContainer(for: Item.self, inMemory: true)
}
