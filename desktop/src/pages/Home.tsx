import { Link } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { useFamily } from '@/contexts/FamilyContext';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import Layout from '@/components/Layout';

export default function HomePage() {
  const { user } = useAuth();
  const { activeFamily } = useFamily();

  return (
    <Layout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Welcome, {user?.name || user?.email}!</h1>
          <p className="text-muted-foreground">Manage your family workspace</p>
        </div>

        {activeFamily && (
          <Card>
            <CardHeader>
              <CardTitle>Active Workspace</CardTitle>
              <CardDescription>Currently viewing: {activeFamily.name}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Role: {activeFamily.role === 'owner' ? 'Owner' : 'Member'}
              </p>
            </CardContent>
          </Card>
        )}

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Lists</CardTitle>
              <CardDescription>Manage your task lists</CardDescription>
            </CardHeader>
            <CardContent>
              <Link to="/lists">
                <Button className="w-full">View Lists</Button>
              </Link>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Notes</CardTitle>
              <CardDescription>Store your notes and files</CardDescription>
            </CardHeader>
            <CardContent>
              <Link to="/notes">
                <Button className="w-full">View Notes</Button>
              </Link>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Birthdays</CardTitle>
              <CardDescription>Track family birthdays</CardDescription>
            </CardHeader>
            <CardContent>
              <Link to="/birthdays">
                <Button className="w-full">View Birthdays</Button>
              </Link>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Workspaces</CardTitle>
            <CardDescription>Manage your workspaces</CardDescription>
          </CardHeader>
          <CardContent>
            <Link to="/families">
              <Button variant="outline" className="w-full">Manage Workspaces</Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

