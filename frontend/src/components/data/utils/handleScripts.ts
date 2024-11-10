// utils/handleScripts.ts
import type { MarketDataType } from '@/types/marketData'
import type { ScriptInfo } from '../interfaces/ScriptInfo'

interface ScriptUploadData {
  name: string
  dataType: MarketDataType
  content: string
}

interface HandleScriptsProps {
  setScripts: React.Dispatch<React.SetStateAction<ScriptInfo[]>>
  onError?: (error: Error) => void
}

export const handleScripts = ({ setScripts, onError }: HandleScriptsProps) => {
  const handleViewCode = async (scriptId: string) => {
    try {
      const code = await window.go.main.App.GetScriptContent(scriptId)
      // Handle the code as needed, e.g., set in state
      // setSelectedScriptCode(code)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleUploadScript = async (scriptData: ScriptUploadData) => {
    try {
      await window.go.main.App.UploadScript(scriptData)
      // Refresh scripts list
      // setScripts(updatedScripts)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleRunScript = async (scriptId: string) => {
    try {
      await window.go.main.App.RunScript(scriptId)
      // Refresh scripts list
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleStopScript = async (scriptId: string) => {
    try {
      await window.go.main.App.StopScript(scriptId)
      // Refresh scripts list
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleDeleteScript = async (scriptId: string) => {
    if (confirm('Are you sure you want to delete this script?')) {
      try {
        await window.go.main.App.DeleteScript(scriptId)
        // Refresh scripts list
      } catch (error) {
        onError?.(error as Error)
      }
    }
  }

  const handleInstallScript = async (scriptId: string) => {
    try {
      await window.go.main.App.InstallScript(scriptId)
      // Refresh scripts list
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleUninstallScript = async (scriptId: string) => {
    try {
      await window.go.main.App.UninstallScript(scriptId)
      // Refresh scripts list
    } catch (error) {
      onError?.(error as Error)
    }
  }

  return {
    handleViewCode,
    handleUploadScript,
    handleRunScript,
    handleStopScript,
    handleDeleteScript,
    handleInstallScript,
    handleUninstallScript,
  }
}
