import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:9090',
})

api.interceptors.request.use((config) => {
  //localStorage.getItem('token')
  const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiODA1MmE5NjQtYTg4ZC00ZjE3LWE4MWUtMDhmYTIxN2RhZWJlIiwicm9sZSI6ImFkbWluIiwiZXhwIjoxNzc3ODQ0MDI5LCJpYXQiOjE3Nzc3NTc2Mjl9.FK_zcw_ujBat9d8r4yFG3ouhhg4v7MVZnChdWdSK5TY"
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export default api