import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import Layout from '@/components/Layout';
import { Plus } from 'lucide-react';

interface Note {
  id: string;
  title: string;
  contentType: 'text' | 'link' | 'copy_text' | 'file';
  content?: string;
  fileUrl?: string;
}

export default function NotesPage() {
  const { activeFamily } = useFamily();
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    if (activeFamily) {
      loadNotes();
    }
  }, [activeFamily]);

  const loadNotes = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.get<Note[]>(`/notes?familyId=${activeFamily.id}`);
      setNotes(data);
    } catch (error) {
      console.error('Failed to load notes:', error);
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
            <h1 className="text-3xl font-bold">Notes</h1>
            <p className="text-muted-foreground">Store your notes and files</p>
          </div>
          <Button onClick={() => navigate('/notes/create')}>
            <Plus className="mr-2 h-4 w-4" />
            Create Note
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-12">Loading...</div>
        ) : notes.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-muted-foreground">No notes yet. Create one to get started!</p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {notes.map((note) => (
              <Card key={note.id} className="cursor-pointer hover:shadow-md transition-shadow">
                <CardHeader>
                  <CardTitle>{note.title}</CardTitle>
                  <CardDescription className="uppercase">{note.contentType}</CardDescription>
                </CardHeader>
                <CardContent>
                  {note.content && (
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {note.content}
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

