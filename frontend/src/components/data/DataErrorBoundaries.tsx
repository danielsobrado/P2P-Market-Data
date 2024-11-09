import React, { Component, ErrorInfo, useEffect, useState } from 'react'
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { 
  RefreshCw, 
  AlertCircle, 
  Bug, 
  FileWarning,
  Network,
  Database
} from 'lucide-react'

interface ErrorDetails {
  title: string
  description: string
  icon: React.ReactNode
  action?: {
    label: string
    handler: () => void
  }
}

// Specific error types
export class NetworkError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'NetworkError'
  }
}

export class DataValidationError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'DataValidationError'
  }
}

export class DatabaseError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'DatabaseError'
  }
}

// Enhanced error boundary with error type handling
interface EnhancedErrorBoundaryProps {
  children: React.ReactNode
  onReset?: () => void
  showDialog?: boolean
  fallbackComponent?: React.ComponentType<{ error: Error; resetError: () => void }>
}

interface EnhancedErrorBoundaryState {
  error: Error | null
  errorInfo: ErrorInfo | null
}

export class EnhancedErrorBoundary extends Component<
  EnhancedErrorBoundaryProps,
  EnhancedErrorBoundaryState
> {
  constructor(props: EnhancedErrorBoundaryProps) {
    super(props)
    this.state = { error: null, errorInfo: null }
  }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo })
    // Log to error reporting service
    console.error('Error caught by boundary:', error, errorInfo)
  }

  getErrorDetails(): ErrorDetails {
    const { error } = this.state
    if (!error) return {
      title: 'Unknown Error',
      description: 'An unexpected error occurred.',
      icon: <AlertCircle className="h-5 w-5" />
    }

    switch (error.name) {
      case 'NetworkError':
        return {
          title: 'Network Error',
          description: 'Failed to connect to the network. Please check your connection.',
          icon: <Network className="h-5 w-5" />,
          action: {
            label: 'Retry Connection',
            handler: () => window.location.reload()
          }
        }
      case 'DataValidationError':
        return {
          title: 'Data Validation Error',
          description: 'The data received was invalid or corrupted.',
          icon: <FileWarning className="h-5 w-5" />,
          action: {
            label: 'Refresh Data',
            handler: () => this.resetError()
          }
        }
      case 'DatabaseError':
        return {
          title: 'Database Error',
          description: 'Failed to access the data store.',
          icon: <Database className="h-5 w-5" />,
          action: {
            label: 'Try Again',
            handler: () => this.resetError()
          }
        }
      default:
        return {
          title: 'Application Error',
          description: error.message || 'An unexpected error occurred.',
          icon: <Bug className="h-5 w-5" />,
          action: {
            label: 'Reset',
            handler: () => this.resetError()
          }
        }
    }
  }

  resetError = () => {
    this.setState({ error: null, errorInfo: null })
    this.props.onReset?.()
  }

  render() {
    if (this.state.error) {
      const { title, description, icon, action } = this.getErrorDetails()

      if (this.props.fallbackComponent) {
        const FallbackComponent = this.props.fallbackComponent
        return <FallbackComponent error={this.state.error} resetError={this.resetError} />
      }

      if (this.props.showDialog) {
        return (
          <ErrorDialog
            title={title}
            description={description}
            icon={icon}
            error={this.state.error}
            errorInfo={this.state.errorInfo}
            onReset={action?.handler || this.resetError}
            actionLabel={action?.label}
          />
        )
      }

      return (
        <Alert variant="destructive">
          <div className="flex items-center gap-2">
            {icon}
            <div>
              <AlertTitle>{title}</AlertTitle>
              <AlertDescription className="mt-2 space-y-4">
                <p>{description}</p>
                {action && (
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={action.handler}
                  >
                    <RefreshCw className="h-4 w-4 mr-2" />
                    {action.label}
                  </Button>
                )}
              </AlertDescription>
            </div>
          </div>
        </Alert>
      )
    }

    return this.props.children
  }
}

// Error Dialog Component
interface ErrorDialogProps {
  title: string
  description: string
  icon: React.ReactNode
  error: Error
  errorInfo: ErrorInfo | null
  onReset: () => void
  actionLabel?: string
}

function ErrorDialog({
  title,
  description,
  icon,
  error,
  errorInfo,
  onReset,
  actionLabel = 'Try Again'
}: ErrorDialogProps) {
  const [isOpen, setIsOpen] = useState(true)
  const [showDetails, setShowDetails] = useState(false)

  useEffect(() => {
    if (!isOpen) {
      onReset()
    }
  }, [isOpen, onReset])

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-2">
            {icon}
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription className="mt-2">
            {description}
          </DialogDescription>
        </DialogHeader>

        <div className="mt-4">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowDetails(!showDetails)}
          >
            {showDetails ? 'Hide' : 'Show'} Technical Details
          </Button>

          {showDetails && (
            <ScrollArea className="mt-4 h-[200px] rounded border p-4">
              <div className="space-y-2 font-mono text-sm">
                <p className="font-bold">Error: {error.name}</p>
                <p>{error.message}</p>
                {errorInfo && (
                  <pre className="mt-2 whitespace-pre-wrap">
                    {errorInfo.componentStack}
                  </pre>
                )}
              </div>
            </ScrollArea>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setIsOpen(false)}>
            Cancel
          </Button>
          <Button onClick={onReset}>
            <RefreshCw className="h-4 w-4 mr-2" />
            {actionLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Specialized Error Boundaries
export function DataLoadingErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <EnhancedErrorBoundary
      onReset={() => {
        // Perform any necessary cleanup or re-initialization
        window.go.main.App.ResetDataConnection()
      }}
      fallbackComponent={({ error, resetError }) => (
        <div className="p-4 space-y-4">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Failed to load data</AlertTitle>
            <AlertDescription>
              {error.message}
              <Button 
                variant="outline" 
                size="sm" 
                className="mt-2"
                onClick={resetError}
              >
                <RefreshCw className="h-4 w-4 mr-2" />
                Retry Loading
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      )}
    >
      {children}
    </EnhancedErrorBoundary>
  )
}

export function DataProcessingErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <EnhancedErrorBoundary
      showDialog
      onReset={() => {
        // Clear any cached data and reset processing state
        window.go.main.App.ResetDataProcessing()
      }}
    >
      {children}
    </EnhancedErrorBoundary>
  )
}

export function NetworkErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <EnhancedErrorBoundary
      onReset={() => {
        // Retry network connection
        window.go.main.App.RetryConnection()
      }}
    >
      {children}
    </EnhancedErrorBoundary>
  )
}

// Helper function to wrap components with multiple error boundaries
export function withErrorBoundaries<P extends object>(
  WrappedComponent: React.ComponentType<P>
) {
  return function WithErrorBoundaries(props: P) {
    return (
      <NetworkErrorBoundary>
        <DataLoadingErrorBoundary>
          <DataProcessingErrorBoundary>
            <WrappedComponent {...props} />
          </DataProcessingErrorBoundary>
        </DataLoadingErrorBoundary>
      </NetworkErrorBoundary>
    )
  }
}

// Usage example:
// const SafeDataComponent = withErrorBoundaries(DataComponent)