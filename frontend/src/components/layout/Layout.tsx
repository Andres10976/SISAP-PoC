import { Header } from "./Header";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <Header />
      <main className="mx-auto max-w-screen-2xl px-6 py-6 flex flex-col gap-6 h-[calc(100vh-4rem)]">
        {children}
      </main>
    </div>
  );
}
