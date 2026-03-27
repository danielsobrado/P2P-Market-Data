import { useEffect, useState } from "react"
import { DataContainer } from "@/components/data/DataContainer"
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

function App() {
  const [isConnected, setIsConnected] = useState(false)
  const [activeView, setActiveView] = useState('data')

  // Check connection status using ServerStatus so the indicator reflects actual backend state
  useEffect(() => {
    const checkConnection = async () => {
      try {
        if (window?.go?.main?.App?.GetServerStatus) {
          const status = await window.go.main.App.GetServerStatus()
          setIsConnected(Boolean(status?.running && status?.databaseConnected))
          return
        }
        setIsConnected(false)
      } catch (error) {
        console.error('Connection check failed:', error)
        setIsConnected(false)
      }
    }
    checkConnection()
    const interval = setInterval(checkConnection, 5000)
    return () => clearInterval(interval)
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
              <div className="flex items-center gap-1">
                <div className={cn(
                  "h-2 w-2 rounded-full",
                  isConnected ? "bg-green-500" : "bg-red-500"
                )} />
                <span className="text-xs text-muted-foreground">
                  {isConnected ? "Connected" : "Disconnected"}
                </span>
              </div>
            </div>
          </div>
        </header>
        
          <main>
            {activeView === 'data' && (
            <DataContainer />
          )}
          {activeView === 'peers' && <PeerManagement />}
        </main>
      </div>
    </ThemeProvider>
  )
}

export default App
