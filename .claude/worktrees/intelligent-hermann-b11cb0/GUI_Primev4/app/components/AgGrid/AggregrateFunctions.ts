const check_box_aggregrate = (params: any) => {
    const values = params.values;
    const allTrue = values.every((value:boolean) => value === true);
    const allFalse = values.every((value:boolean) => value === false);
    return allTrue ? true : allFalse ? false : null; // Return null for mixed state
  };
export default  check_box_aggregrate 