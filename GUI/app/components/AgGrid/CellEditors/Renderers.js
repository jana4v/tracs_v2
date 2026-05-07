function CellRenderer(params) {
  const error = isNaN(params.value) // replace with your validation logic
  if (error) {
    return {
      vCell: { title: 'Input should be a number', innerHTML: params.value },
    }
  }
  else {
    return params.value
  }
}
