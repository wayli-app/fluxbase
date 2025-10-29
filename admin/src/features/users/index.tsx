import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'

export function Users() {
  const navigate = useNavigate()

  // Redirect to tables page with auth.users pre-selected
  useEffect(() => {
    navigate({ to: '/tables', search: { table: 'auth.users' } })
  }, [navigate])

  return null
}
