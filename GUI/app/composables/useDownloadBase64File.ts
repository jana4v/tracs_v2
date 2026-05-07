export function downloadBase64File(options: {
  filename: string
  content: string
  mime_type: string
}): void {
  const { filename, content, mime_type } = options
  const link = document.createElement('a')
  link.href = `data:${mime_type};base64,${content}`
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
