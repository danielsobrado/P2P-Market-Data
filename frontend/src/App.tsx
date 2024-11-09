import { SetStateAction, useEffect, useState } from "react"
import DataManagement from "@/components/data/DataManagement"
import PeerManagement from "@/components/peer/PeerManagement"
import { ThemeProvider } from "./components/theme-provider"
import { cn } from "@/lib/utils"
import ModeToggle from "@/components/mode-toggle"
import { Button } from "@/components/ui/button"
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
} from "@/components/ui/navigation-menu"
import { DataRequest } from "./types/marketData"

function App() {
  const [isConnected, setIsConnected] = useState(false)
  const [activeView, setActiveView] = useState('data')

  useEffect(() => {
    // Check P2P connection status
    const checkConnection = async () => {
      try {
        const status = await window.go.main.App.CheckConnection()
        setIsConnected(status)
      } catch (error) {
        console.error('Connection check failed:', error)
      }
    }
    checkConnection()
  }, [])

  return (
    <ThemeProvider defaultTheme="system" storageKey="ui-theme">
      <div className="min-h-screen bg-background">
        <header className="border-b">
          <div className="container flex items-center justify-between py-4">
            <h1 className="text-2xl font-bold">P2P Market Data</h1>
            <div className="flex items-center gap-4">
              <NavigationMenu>
                <NavigationMenuList>
                  <NavigationMenuItem>
                    <NavigationMenuTrigger>Navigation</NavigationMenuTrigger>
                    <NavigationMenuContent>
                      <Button
                        variant="ghost"
                        className={cn(
                          "w-full justify-start",
                          activeView === 'data' && "bg-accent"
                        )}
                        onClick={() => setActiveView('data')}
                      >
                        Market Data
                      </Button>
                      <Button
                        variant="ghost"
                        className={cn(
                          "w-full justify-start",
                          activeView === 'peers' && "bg-accent"
                        )}
                        onClick={() => setActiveView('peers')}
                      >
                        Peer Management
                      </Button>
                    </NavigationMenuContent>
                  </NavigationMenuItem>
                </NavigationMenuList>
              </NavigationMenu>
              <ModeToggle />
              <div className={cn(
                "h-2 w-2 rounded-full",
                isConnected ? "bg-green-500" : "bg-red-500"
              )} />
            </div>
          </div>
        </header>
        
        <main className="container py-8">
          {activeView === 'data' && <DataManagement sources={[]} transfers={[]} searchResults={[]} onSearch={function (request: DataRequest): Promise<void> {
            throw new Error("Function not implemented.")
          } } onRequestData={function (peerId: string, request: DataRequest): Promise<void> {
            throw new Error("Function not implemented.")
          } } onClearSearch={function (): void {
            throw new Error("Function not implemented.")
          } } isLoading={false} setPollingEnabled={function (value: SetStateAction<boolean>): void {
            throw new Error("Function not implemented.")
          } } />}
          {activeView === 'peers' && <PeerManagement />}
        </main>
      </div>
    </ThemeProvider>
  )
}

export default App
