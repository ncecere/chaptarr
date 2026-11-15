import { Compass } from 'lucide-react'
import { Link } from 'react-router-dom'
import { EmptyState } from '../components/States'
import { useDocumentTitle } from '../lib/hooks'

export function NotFoundPage() {
  useDocumentTitle('Not found')
  return (
    <EmptyState
      icon={<Compass />}
      title="Page not found"
      description="That page doesn't exist."
      action={
        <Link to="/" className="text-accent hover:underline">
          Back to the library
        </Link>
      }
    />
  )
}
