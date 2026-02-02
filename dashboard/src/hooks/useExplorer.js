import { useState, useCallback } from 'react';
import axios from 'axios';

const useExplorer = (token) => {
    const [explorerPath, setExplorerPath] = useState('');
    const [explorerFiles, setExplorerFiles] = useState([]);

    const fetchExplorer = useCallback(async (path = '') => {
        if (!token) return;
        try {
            const config = { headers: { 'X-API-Key': token } };
            const res = await axios.get(`/api/explorer/remote?path=${encodeURIComponent(path)}`, config);

            const normalizedData = (res.data || []).map(item => ({
                name: item.Name,
                displayName: item.Name,
                isDir: item.IsDir,
                size: item.Size,
                time: item.ModTime,
                status: 'cloud'
            }));

            setExplorerFiles(normalizedData);
            setExplorerPath(path);
        } catch (err) {
            console.error("Explorer error:", err);
        }
    }, [token]);

    const getExternalLink = async (name) => {
        try {
            const config = { headers: { 'X-API-Key': token } };
            const path = explorerPath ? `${explorerPath}/${name}` : name;
            const res = await axios.get(`/api/explorer/remote/link?path=${encodeURIComponent(path)}`, config);
            if (res.data.link) {
                window.open(res.data.link, '_blank');
            } else {
                alert('No public link available');
            }
        } catch (err) { alert('Failed to generate cloud link'); }
    };

    const deleteFile = async (name) => {
        if (confirm(`Permanently delete ${name}?`)) {
            try {
                const config = { headers: { 'X-API-Key': token } };
                const path = explorerPath ? `${explorerPath}/${name}` : name;
                await axios.delete(`/api/explorer/remote?path=${encodeURIComponent(path)}`, config);
                fetchExplorer(explorerPath);
            } catch (err) { alert('Failed to delete cloud item'); }
        }
    };

    return { explorerPath, setExplorerPath, explorerFiles, fetchExplorer, getExternalLink, deleteFile };
};

export default useExplorer;
