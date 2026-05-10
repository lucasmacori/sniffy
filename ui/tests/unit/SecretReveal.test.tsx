import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SecretReveal } from '@/components/SecretReveal'

describe('SecretReveal', () => {
  it('renders masked preview by default', () => {
    render(<SecretReveal secret="my-super-secret-key" />)
    expect(screen.getByText(/my-super•••/)).toBeInTheDocument()
  })

  it('reveals secret on click', () => {
    render(<SecretReveal secret="my-super-secret-key" />)
    const button = screen.getByRole('button', { name: /reveal/i })
    fireEvent.click(button)
    expect(screen.getByText('my-super-secret-key')).toBeInTheDocument()
  })

  it('hides secret on second click', () => {
    render(<SecretReveal secret="my-super-secret-key" />)
    const button = screen.getByRole('button')
    fireEvent.click(button)
    fireEvent.click(button)
    expect(screen.getByText(/my-super•••/)).toBeInTheDocument()
  })

  it('respects maxPreview prop', () => {
    render(<SecretReveal secret="abcdefgh" maxPreview={3} />)
    expect(screen.getByText('abc•••')).toBeInTheDocument()
  })
})
