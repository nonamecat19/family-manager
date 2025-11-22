import { Redirect } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useAuth } from '@/contexts/AuthContext';

export default function Index() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return null;
  }

  return (
    <>
      <StatusBar style='auto' />
      {!isAuthenticated ? (
        <Redirect href="/auth/login" />
      ) : (
        <Redirect href="/(tabs)" />
      )}
    </>
  );
}

