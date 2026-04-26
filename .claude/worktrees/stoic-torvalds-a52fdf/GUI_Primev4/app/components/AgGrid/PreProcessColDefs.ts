import CellStyles from './CellStyles/Validators';

const PrepProcessColDefs = (colDefs: Record<string, any>[]) => {
  // Now map the cellStyle string to the corresponding function
  const processedColumnDefs = colDefs.map((colDef) => {
    const styleFunction = CellStyles[colDef.cellStyle];

    if (styleFunction) {
      // If the function exists in the map, use it
      colDef = { ...colDef, cellStyle: styleFunction };
    }

    return colDef;
  });
  return processedColumnDefs;
};

export default PrepProcessColDefs;
