import React, { Component, ErrorInfo } from 'react'
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { RefreshCw, AlertCircle } from 'lucide-react'

interface Props {
  children: React.ReactNode
  fallback?: React.ReactNode
  onReset?: () => void
}

interface State {
  hasError: boolean
  error?: Error
}

export class DataErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Data component error:', error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined })
    this.props.onReset?.()
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription className="space-y-4">
            <p>An error occurred while loading the data component.</p>
            <p className="text-sm font-mono">{this.state.error?.message}</p>
            <Button 
              variant="outline" 
              size="sm" 
              onClick={this.handleReset}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Try Again
            </Button>
          </AlertDescription>
        </Alert>
      )
    }

    return this.props.children
  }
}

// Higher-order component for easy wrapping
export function withDataErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  fallback?: React.ReactNode
) {
  return function WithErrorBoundary(props: P) {
    return (
      <DataErrorBoundary fallback={fallback}>
        <WrappedComponent {...props} />
      </DataErrorBoundary>
    )
  }
}