import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import Layout from '@/components/Layout';
import { Plus } from 'lucide-react';

interface Birthday {
  id: string;
  name: string;
  surname?: string;
  dateOfBirth: string;
  daysUntil: number;
  nextBirthday: string;
}

export default function BirthdaysPage() {
  const { activeFamily } = useFamily();
  const [birthdays, setBirthdays] = useState<Birthday[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    if (activeFamily) {
      loadBirthdays();
    }
  }, [activeFamily]);

  const loadBirthdays = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.get<Birthday[]>(`/birthdays?familyId=${activeFamily.id}`);
      setBirthdays(data);
    } catch (error) {
      console.error('Failed to load birthdays:', error);
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
            <h1 className="text-3xl font-bold">Birthdays</h1>
            <p className="text-muted-foreground">Track family birthdays</p>
          </div>
          <Button onClick={() => navigate('/birthdays/create')}>
            <Plus className="mr-2 h-4 w-4" />
            Add Birthday
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-12">Loading...</div>
        ) : birthdays.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-muted-foreground">No birthdays yet. Add one to get started!</p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {birthdays.map((birthday) => (
              <Card key={birthday.id} className="cursor-pointer hover:shadow-md transition-shadow">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle>
                      {birthday.name} {birthday.surname || ''}
                    </CardTitle>
                    <span className="text-sm font-semibold text-primary">
                      {birthday.daysUntil === 0 ? 'Today!' : `${birthday.daysUntil} days`}
                    </span>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">
                    {new Date(birthday.dateOfBirth).toLocaleDateString('en-US', {
                      month: 'long',
                      day: 'numeric',
                    })}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </Layout>
  );
}

