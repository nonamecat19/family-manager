import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import Layout from '@/components/Layout';
import { Plus } from 'lucide-react';

interface List {
  id: string;
  title: string;
  description?: string;
  completed: boolean;
  dueDate?: string;
}

export default function ListsPage() {
  const { activeFamily } = useFamily();
  const [lists, setLists] = useState<List[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    if (activeFamily) {
      loadLists();
    }
  }, [activeFamily]);

  const loadLists = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.get<List[]>(`/lists?familyId=${activeFamily.id}`);
      setLists(data);
    } catch (error) {
      console.error('Failed to load lists:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!activeFamily) {
    return (
      <Layout>
        <div className="text-center py-12">
          <p className="text-muted-foreground">Please select a workspace</p>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Lists</h1>
            <p className="text-muted-foreground">Manage your task lists</p>
          </div>
          <Button onClick={() => navigate('/lists/create')}>
            <Plus className="mr-2 h-4 w-4" />
            Create List
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-12">Loading...</div>
        ) : lists.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-muted-foreground">No lists yet. Create one to get started!</p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {lists.map((list) => (
              <Card key={list.id} className="cursor-pointer hover:shadow-md transition-shadow">
                <CardHeader>
                  <CardTitle>{list.title}</CardTitle>
                  {list.description && (
                    <CardDescription>{list.description}</CardDescription>
                  )}
                </CardHeader>
                <CardContent>
                  {list.dueDate && (
                    <p className="text-sm text-muted-foreground">
                      Due: {new Date(list.dueDate).toLocaleDateString()}
                    </p>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </Layout>
  );
}

