import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import Layout from '@/components/Layout';
import { Plus } from 'lucide-react';

export default function FamiliesPage() {
  const { families, activeFamily, setActiveFamily, refreshFamilies } = useFamily();
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSelectFamily = async (family: any) => {
    try {
      await setActiveFamily(family);
      navigate('/');
    } catch (error: any) {
      console.error('Failed to switch workspace:', error);
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Workspaces</h1>
            <p className="text-muted-foreground">Manage your workspaces</p>
          </div>
          <Button onClick={() => navigate('/families/create')}>
            <Plus className="mr-2 h-4 w-4" />
            Create Workspace
          </Button>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {families.map((family) => (
            <Card
              key={family.id}
              className={activeFamily?.id === family.id ? 'border-primary' : ''}
            >
              <CardHeader>
                <CardTitle>{family.name}</CardTitle>
                <CardDescription>
                  {family.role === 'owner' ? 'Owner' : 'Member'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {activeFamily?.id === family.id ? (
                  <div className="text-sm text-primary font-medium">Active</div>
                ) : (
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => handleSelectFamily(family)}
                  >
                    Switch to this workspace
                  </Button>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </Layout>
  );
}

