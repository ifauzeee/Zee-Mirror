import { useState, useEffect, useCallback } from 'react';
import axios from 'axios';

const useTasks = (token) => {
    const [tasks, setTasks] = useState([]);

    const fetchTasks = useCallback(async () => {
        if (!token) return;
        try {
            const config = { headers: { 'X-API-Key': token } };
            const res = await axios.get('/api/tasks', config);
            setTasks(res.data || []);
        } catch (err) {
            console.error(err);
        }
    }, [token]);

    const cancelTask = async (id) => {
        if (confirm('Cancel this task?')) {
            try {
                const config = { headers: { 'X-API-Key': token } };
                await axios.delete(`/api/tasks?id=${id}`, config);
                fetchTasks();
            } catch (err) { console.error(err); }
        }
    };

    return { tasks, fetchTasks, cancelTask, setTasks };
};

export default useTasks;
