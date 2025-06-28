import { useState, useEffect } from "react";
import axios from "axios";
import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import 'boxicons/css/boxicons.min.css';
import UploadCourseDialog from "../component/UploadCourse";
import CustomButton from "../../compounds/button";
import DataTable from "../../compounds/tables/data-table";
import '../../index.css';

const API_BASE_URL = "http://localhost:8080/api";

const EditInput = ({ value, onChange, name }) => (
  <input
    type="text"
    name={name}
    value={value || ''}
    onChange={onChange}
    className="border rounded px-2 py-1 w-full shadow-sm border-gray-300 bg-white"
  />
);

const EditSelect = ({ value, onChange, name, options }) => (
  <select
    name={name}
    value={value || ''}
    onChange={onChange}
    className="border rounded px-2 py-1 w-full shadow-sm border-gray-300 bg-white"
  >
    <option value="" disabled>Select...</option>
    {options.map(option => (
      <option key={option.value} value={option.value}>{option.label}</option>
    ))}
  </select>
);

function BasicExample() {
  const [userData, setUserData] = useState([]);
  const [departments, setDepartments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showUploadDialog, setShowUploadDialog] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [editedUser, setEditedUser] = useState({});

  useEffect(() => {
    // Fetch both users and departments data when the component mounts
    const fetchInitialData = async () => {
      setLoading(true);
      try {
        const [usersRes, deptsRes] = await Promise.all([
          axios.get(`${API_BASE_URL}/users`),
          axios.get(`${API_BASE_URL}/departments`)
        ]);
        setUserData(usersRes.data || []);
        setDepartments(deptsRes.data || []);
      } catch (err) {
        console.error("Failed to fetch initial data:", err);
        toast.error("Failed to fetch data from server.");
      } finally {
        setLoading(false);
      }
    };
    fetchInitialData();
  }, []);

  const fetchUserData = () => {
    // This function can be used to refresh just the user list after an update/upload
    axios.get(`${API_BASE_URL}/users`)
      .then((res) => {
        setUserData(res.data || []);
      })
      .catch((err) => {
        console.error("Failed to fetch user data:", err);
        toast.error("Failed to refresh user data.");
      });
  };

  const handleOpenUploadDialog = () => setShowUploadDialog(true);
  const handleCloseUploadDialog = () => setShowUploadDialog(false);

  const handleEditClick = (user) => {
    setEditingId(user.id);
    setEditedUser({ ...user });
  };

  const handleCancelClick = () => {
    setEditingId(null);
    setEditedUser({});
  };

  const handleEditChange = (e) => {
    const { name, value } = e.target;
    // Ensure dept value is an integer
    const processedValue = name === "dept" ? parseInt(value, 10) : value;
    setEditedUser(prev => ({ ...prev, [name]: processedValue }));
  };

  const handleUpdateClick = (id) => {
    axios.put(`${API_BASE_URL}/users/${id}`, editedUser)
      .then(() => {
        toast.success("User updated successfully!");
        setEditingId(null);
        fetchUserData(); // Refresh user data
      })
      .catch(err => {
        console.error("Failed to update user:", err);
        const errorMsg = err.response?.data?.error || err.message;
        toast.error(`Update failed: ${errorMsg}`);
      });
  };

  const handleDeleteClick = (id, userName) => {
    if (window.confirm(`Are you sure you want to delete the user: ${userName}?`)) {
      axios.delete(`${API_BASE_URL}/users/${id}`)
        .then(() => {
          toast.info("User deleted successfully!");
          fetchUserData(); // Refresh user data
        })
        .catch(err => {
          console.error("Failed to delete user:", err);
          const errorMsg = err.response?.data?.error || err.message;
          toast.error(`Delete failed: ${errorMsg}`);
        });
    }
  };

  // Headers and fields for the data table
  const headers = ["ID", "Name", "Email", "Dept", "Year", "Degree"];
  const fields = ["id", "name", "email", "deptName", "year", "degree"];

  // Dynamic dropdown options derived from state
  const dropdownOptions = {
    dept: departments.map(d => ({ label: d.dept_short, value: d.id })),
    degree: [
      { label: "UG", value: "UG" },
      { label: "PG", value: "PG" },
      { label: "MBA", value: "MBA" },
      { label: "PHD", value: "PHD" },
    ],
  };

  // Custom renderers for displaying data and edit fields
  const customRenderers = fields.reduce((acc, field) => {
    acc[field] = (user) => {
      if (editingId === user.id) {
        if (field === "deptName") { // Use 'deptName' for check but edit 'dept'
          return (
            <EditSelect
              name="dept"
              value={editedUser.dept}
              onChange={handleEditChange}
              options={dropdownOptions.dept}
            />
          );
        }
        if (field === "degree") {
          return (
            <EditSelect
              name="degree"
              value={editedUser.degree}
              onChange={handleEditChange}
              options={dropdownOptions.degree}
            />

          );
        }
        // Default text input for other fields
        return (
          <EditInput
            name={field}
            value={editedUser[field]}
            onChange={handleEditChange}
          />
        );
      }
      // For Dept, we now display the deptName directly from the user object
      return user[field];
    };
    return acc;
  }, {});
  
  // Custom renderer for the actions column
  const actionsRenderer = (user) => {
    return editingId === user.id ? (
      <div className="flex items-center gap-2 justify-center">
        <CustomButton label="Update" onClick={() => handleUpdateClick(user.id)} others="bg-green-500 hover:bg-green-600 px-3 py-1 text-white rounded-md shadow-sm" />
        <CustomButton label="Cancel" onClick={handleCancelClick} others="bg-gray-500 hover:bg-gray-600 px-3 py-1 text-white rounded-md shadow-sm" />
      </div>
    ) : (
      <div className="flex justify-center items-center gap-2">
        <button className="text-blue-500 hover:text-blue-700"
        style={{ color: "black" }}
         onClick={() => handleEditClick(user)}>
          <i className="bx bx-edit bx-sm bx-tada-hover"></i>
        </button>
        <div className="h-10 w-[1.5px] bg-gray-300"></div>
        <button className="text-red-500 hover:text-red-700"
        style={{ color: "red" }} 
        onClick={() => handleDeleteClick(user.id, user.name)}>
          <i className="bx bx-trash bx-sm bx-tada-hover"></i>
        </button>
      </div>
    );
  };

  return (
    <div style={{ padding: "1rem" }} className="font-inter">
      <ToastContainer position="top-right" autoClose={3000} hideProgressBar={false} newestOnTop={true} />
      {loading ? (
        <div className="flex justify-center items-center h-48 text-lg text-gray-700">Loading users...</div>
      ) : (
        <>
          <DataTable
            title="All Users"
            data={userData}
            headers={headers}
            fields={fields}
            allow_download={true}
            addButton={handleOpenUploadDialog}
            addBtnLabel="Upload Users"
            customRender={customRenderers}
            actions={actionsRenderer}
          />
          <UploadCourseDialog
            open={showUploadDialog}
            handleClose={handleCloseUploadDialog}
            onUploadSuccess={fetchUserData}
          />
        </>
      )}
    </div>
  );
}

export default BasicExample;
