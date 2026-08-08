import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

interface ApiErrorAlertProps {
  message: string
}

export function ApiErrorAlert({ message }: ApiErrorAlertProps) {
  return (
    <Alert variant="destructive">
      <AlertCircle className="size-4" />
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}
