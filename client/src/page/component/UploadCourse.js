import React, { useState } from "react";
import axios from "axios";
import * as XLSX from 'xlsx';
import { toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import CustomDialog from "../../compounds/dialogy"; // Adjust path as per your project structure
import CustomButton from "../../compounds/button";   // Adjust path as per your project structure
import UsersampleFile from "../../assets/img/files/student_data.xlsx"; // Using the correct sample file path

function UploadCourse({ open, handleClose, onUploadSuccess }) {
  const [csvData, setCsvData] = useState(null);
  const [fileName, setFileName] = useState("");
  const [errMsg, setErrMsg] = useState("");
  const [notUpdated, setNotUpdated] = useState([]); // Keep for potential future use
  const [processing, setProcessing] = useState(false);

  const readExcel = (file) => {
    if (!file) {
      setCsvData(null);
      setFileName("");
      setErrMsg("No file selected.");
      return;
    }

    if (!file.name.match(/\.(xlsx|xls)$/)) {
        setErrMsg("Invalid file format. Please upload an .xlsx or .xls file.");
        setCsvData(null);
        setFileName("");
        return;
    }

    setFileName(file.name);
    setErrMsg("");
    setNotUpdated([]);
    setProcessing(true);

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const data = new Uint8Array(e.target.result);
        const workbook = XLSX.read(data, { type: 'array' });
        const sheetName = workbook.SheetNames[0];
        const worksheet = workbook.Sheets[sheetName];
        const csv = XLSX.utils.sheet_to_csv(worksheet);
        setCsvData(csv);
        console.log("File processed and converted to CSV.");
      } catch (error) {
        setErrMsg("Error reading or converting the file.");
        console.error("Error processing file:", error);
      } finally {
        setProcessing(false);
      }
    };
    reader.onerror = (error) => {
      setErrMsg("Error reading file.");
      console.error("Error reading file:", error);
      setProcessing(false);
    };
    reader.readAsArrayBuffer(file);
  };

  const insertData = () => {
    if (!csvData) {
      setErrMsg("Please select and process a file first.");
      return;
    }
    
    setProcessing(true);
    const formData = new FormData();
    const csvBlob = new Blob([csvData], { type: 'text/csv' });
    formData.append("file", csvBlob, "upload.csv");

    axios.post("http://localhost:8080/api/upload-users", formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    })
    .then((res) => {
      if (res.status === 202 || (res.data.errors && res.data.errors.length > 0)) {
        toast.warn(<div>Users uploaded with some errors. <br/> See console for details.</div>, { autoClose: 5000 });
        console.warn("Processing errors:", res.data.errors);
        // If your backend ever sends back a 'notUpdated' list, this will handle it
        if (res.data.notUpdated && res.data.notUpdated.length > 0) {
            setNotUpdated(res.data.notUpdated);
        }
      } else {
        toast.success("Users uploaded successfully!");
      }
      handleDialogClose();
      onUploadSuccess();
    })
    .catch((err) => {
      console.error("Failed to upload file:", err);
      const errorMsg = err.response?.data?.message || "File upload failed. Please try again.";
      setErrMsg(errorMsg);
      toast.error(errorMsg);
    }).finally(() => {
        setProcessing(false);
    });
  };
  
  const handleDialogClose = () => {
      handleClose();
      setErrMsg("");
      setCsvData(null);
      setFileName("");
      setNotUpdated([]);
      setProcessing(false);
  };

  const handleDownload = () => {
    if (notUpdated.length > 0) {
      const ws = XLSX.utils.json_to_sheet(notUpdated);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, "NotUpdatedUsers");
      XLSX.writeFile(wb, "not_updated_users.xlsx");
    } else {
      toast.info("There is no data for 'not updated users' to download.");
    }
  };

  return (
    <CustomDialog
      open={open}
      handleClose={handleDialogClose}
      title={"Upload User Data"}
      body={
        <>
          <br />
          <a
            href={UsersampleFile}
            className="flex gap-3 bg-primary w-max p-2 px-6 mt-2 text-white rounded-3xl cursor-pointer"
            download="sample_student_data.xlsx"
          >
            <i className="bx bx-download bx-sm"></i>
            <h3>Download Sample File</h3>
          </a>
          <br />
          <div className="flex flex-col">
            <label style={{ fontSize: 14 }}>Choose File</label>
            <input
              type="file"
              accept=".xlsx, .xls"
              onChange={(e) => readExcel(e.target.files[0])}
                            className="mt-1 p-2 border border-gray-300 rounded"

            />
            {fileName && <p className="text-xs text-gray-600 mt-1">Selected: {fileName}</p>}
          </div>

          {errMsg && (
            <h3 className="text-red-600 font-normal text-sm p-1 rounded mt-2">
              {errMsg}
            </h3>
          )}

          {notUpdated.length > 0 && (
            <>
              <h2 className="mt-5 text-md text-red-500 font-medium">
                Some users were not updated. Kindly download the list and re-upload.
              </h2>
              <div
                onClick={handleDownload}
                className="flex gap-3 bg-primary w-max p-2 px-6 mt-2 text-white rounded-3xl cursor-pointer"
              >
                <i className="bx bx-download bx-sm"></i>
                <h3>Download</h3>
              </div>
            </>
          )}

          <CustomButton
            label={processing ? "Processing..." : "Submit"}
            onClick={insertData}
            others={`mt-4 w-full ${!csvData || processing ? "opacity-50 cursor-not-allowed" : ""}`}
            disabled={!csvData || processing}
          />
        </>
      }
    />
  );
}

export default UploadCourse;
