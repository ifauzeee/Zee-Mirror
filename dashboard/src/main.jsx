import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'
import { PopupProvider } from './context/PopupContext'

ReactDOM.createRoot(document.getElementById('root')).render(
    <React.StrictMode>
        <PopupProvider>
            <App />
        </PopupProvider>
    </React.StrictMode>,
)
