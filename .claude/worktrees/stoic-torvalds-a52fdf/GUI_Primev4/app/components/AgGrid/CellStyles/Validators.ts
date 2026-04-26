
const CellStyles: Record<string, any> = {};

function checkInputForPossibleTelemetryValues(input: string) {
  // Regex for comma separated words
  const commaSepWordsRegex = /^([a-zA-Z0-9.]+\,)*[a-zA-Z0-9.]+$/;

  // Regex for numeric range
  const numericRangeRegex = /^(-?\d+(\.\d+)?\:){2}-?\d+(\.\d+)?$/;

  if (commaSepWordsRegex.test(input)) {
    return false;
  } else if (numericRangeRegex.test(input)) {
    return false;
  } else {
    return true;
  }
}
function checkInputForAllowedTelemetryValues(input: string) {
  // Regex for comma separated words
  const commaSepWordsRegex = /^([a-zA-Z0-9.]+\,)*[a-zA-Z0-9.]+$/;

  // Regex for numeric range
  const numericRangeRegex = /^(-?\d+(\.\d+)?\:){1}-?\d+(\.\d+)?$/;

  if (commaSepWordsRegex.test(input)) {
    return false;
  } else if (numericRangeRegex.test(input)) {
    return false;
  } else {
    return true;
  }
}

CellStyles['telemetry_possible_values'] = (params: any) => {
  if (params.value) {
    console.log(params);
    return (
      checkInputForPossibleTelemetryValues(params.value) && !params.node.group
    );
  } else {
    return true && !params.node.group;
  }
};
CellStyles['telemetry_allowed_values'] = (params: any) => {
  if (params.value) {
    console.log(params.value);
    return (
      checkInputForAllowedTelemetryValues(params.value) && !params.node.group
    );
  } else {
    return true && !params.node.group;
  }
};

CellStyles['atlest_n_characters'] = (char_length: number) => {
  return (params: any) => {
    //console.log(params);
    if (params.value) {
      return String(params.value).length < char_length;
    } else {
      return true && !params.node.group;
    }
  };
};
CellStyles['atlest_n_characters_or_blank'] = (char_length: number) => {
  
  return (params: any) => {
    console.log(params.value);
    if (params.value) {
      return String(params.value).length < char_length;
    } else {
      return false;
    }
  };
};
CellStyles['should_be_numeric'] = (params: any) => {
  if (params.value) {
    return isNaN(params.value) && !params.node.group;
  } else {
    return false;
  }
};
CellStyles['should_be_numeric_not_null'] = (params: any) => {
  if (params.value) {
    return isNaN(params.value) && !params.node.group;
  } else {
    return true && !params.node.group;
  }
};
CellStyles['should_be_positive'] = (params: any) => {
  if (params.value) {
    if (!isNaN(params.value)) {
      return parseFloat(params.value) < 0;
    } else {
      return true;
    }
  }
  return false;
};
CellStyles['should_be_positive_not_null'] = (params: any) => {
  if (params.value) {
    if (!isNaN(params.value)) {
      return parseFloat(params.value) < 0;
    }
  }
  return true && !params.node.group;
};
CellStyles['should_be_negative'] = (params: any) => {
  if (params.value) {
    if (!isNaN(params.value)) {
      return parseFloat(params.value) > 0;
    } else {
      return true;
    }
  }
  return false;
};
CellStyles['should_be_negative_not_null'] = (params: any) => {
  if (params.value) {
    if (!isNaN(params.value)) {
      return parseFloat(params.value) > 0;
    } else {
      return true;
    }
  }
  return true && !params.node.group;
};
CellStyles['atleast_yes_no'] = (params: any) => {
  
};
// const should_be_numeric_pairs_not_null = (params: any) => {
//   if (params.value) {
//     const v = String(params.value).split(',');
//     if (v.length != 2) return true;
//     return isNaN(v[0]) && isNaN(v[1]);
//   } else {
//     return true && !params.node.group;
//   }
// };
// const should_be_numeric_pairs = (params: any) => {
//   if (params.value) {
//     const v = String(params.value).split(',');
//     if (v.length != 2) return true;
//     return isNaN(v[0]) && isNaN(v[1]);
//   } else {
//     return false && !params.node.group;
//   }
// };

// const spurious_data_validator = (params: any) => {
//   if (params.value) {
//     const r =
//       /[\s]*[\(][\s]*[-]?[\d]+[.]*[\d]*[\s]*[,][\s]*[-]?[\d]+[.]*[\d]*[\s]*[\)]/g;
//     const value = String(params.value).replaceAll(';', '');
//     const data = value.split(r);
//     //console.log(data);
//     let err = false;
//     data.forEach((d) => {
//       if (d.length > 0) err = true;
//     });
//     return err;
//   } else {
//     return false && !params.node.group;
//   }
// };

CellStyles['unique_value_in_column'] = (column_name: string) => {
  return (params: any) => {
    if (params.value) {
      let data: any = [];
      params.api.forEachNode((node: any)=> data.push(node.data[column_name]));
      let sameValueCount = data.filter((val: any) => val === params.value).length;
      if(sameValueCount==1){
        params.data.tooltip = "";
      }else{
        params.data.tooltip = "Values in this column should be unique";
      }
      return sameValueCount > 1;
    } else {
      return true && !params.node.group;
    }
  };
};



export default CellStyles;
