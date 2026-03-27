// utils/handleScripts.ts
import type { MarketDataType } from '@/types/marketData'
import type { ScriptInfo } from '../interfaces/ScriptInfo'
import { MarketDataBase } from '../interfaces/MarketDataBase'

interface ScriptUploadData {
  name: string
  dataType: MarketDataType
  content: string
}

interface HandleScriptsProps {
  setScripts: React.Dispatch<React.SetStateAction<ScriptInfo[]>>
  /** Called after any mutation so the scripts list is refreshed from the backend. */
  refreshScripts: () => Promise<void>
  onError?: (error: Error) => void
}

interface App {
  UpdateMarketData: (data: MarketDataBase[]) => Promise<void>
  GetActiveTransfers: () => Promise<DataTransfer[]>
  GetScriptContent: (scriptId: string) => Promise<string>
  UploadScript: (scriptData: ScriptUploadData) => Promise<void>
  RunScript: (scriptId: string) => Promise<void>
  StopScript: (scriptId: string) => Promise<void>
  DeleteScript: (scriptId: string) => Promise<void>
  InstallScript: (scriptId: string) => Promise<void>
  UninstallScript: (scriptId: string) => Promise<void>
}

export const handleScripts = ({ setScripts, refreshScripts, onError }: HandleScriptsProps) => {
  /** Fetches and returns the source code of a script. */
  const handleViewCode = async (scriptId: string): Promise<string | undefined> => {
    try {
      return await window.go.main.App.GetScriptContent(scriptId)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleUploadScript = async (scriptData: ScriptUploadData) => {
    try {
      await window.go.main.App.UploadScript(scriptData)
      await refreshScripts()
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleRunScript = async (scriptId: string) => {
    try {
      await window.go.main.App.RunScript(scriptId)
      await refreshScripts()
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleStopScript = async (scriptId: string) => {
    try {
      await window.go.main.App.StopScript(scriptId)
      await refreshScripts()
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleDeleteScript = async (scriptId: string) => {
    if (confirm('Are you sure you want to delete this script?')) {
      try {
        await window.go.main.App.DeleteScript(scriptId)
        await refreshScripts()
      } catch (error) {
        onError?.(error as Error)
      }
    }
  }

  const handleInstallScript = async (scriptId: string) => {
    try {
      await window.go.main.App.InstallScript(scriptId)
      await refreshScripts()
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleUninstallScript = async (scriptId: string) => {
    try {
      await window.go.main.App.UninstallScript(scriptId)
      await refreshScripts()
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
