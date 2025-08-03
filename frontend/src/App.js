import React, { useState, useEffect } from 'react';
import { LayoutDashboard, Calendar, BarChart2, MessageSquare, Plus, Loader2, Menu } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// The base URLs for our Go backend microservices.
const AUTH_API_BASE_URL = 'http://localhost:8081';
const ACCOUNT_API_BASE_URL = 'http://localhost:8082';
const POST_API_BASE_URL = 'http://localhost:8083';

// Reusable component for displaying an account summary card.
const AccountCard = ({ account }) => (
  <div className="bg-white p-6 rounded-xl shadow-lg transition-transform duration-300 hover:scale-105 transform hover:shadow-2xl flex flex-col items-center text-center">
    <div className="w-20 h-20 rounded-full overflow-hidden mb-4 border-4 border-gray-100">
      <img src={account.profilePic} alt={`${account.username}'s profile`} className="w-full h-full object-cover" />
    </div>
    <h3 className="font-bold text-xl text-gray-800">{account.username}</h3>
    <p className="text-sm text-gray-500 mb-2">{account.platform}</p>
    <p className="text-2xl font-extrabold text-blue-600">{account.followers}</p>
    <p className="text-xs text-gray-400 uppercase tracking-wide">Followers</p>
  </div>
);

// Dashboard component, now fetches accounts from the backend.
const Dashboard = ({ token }) => {
  const [accounts, setAccounts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [analyticsData, setAnalyticsData] = useState([]);

  useEffect(() => {
    const fetchAccounts = async () => {
      try {
        const response = await fetch(`${ACCOUNT_API_BASE_URL}/api/accounts`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });

        if (!response.ok) {
          throw new Error('Failed to fetch accounts. Please log in again.');
        }

        const data = await response.json();
        setAccounts(data);
      } catch (e) {
        console.error('Fetch error:', e);
        setError(e.message);
      } finally {
        setIsLoading(false);
      }
    };
    
    // The analytics endpoint is now on the Post Service.
    const fetchAnalytics = async () => {
      try {
        const response = await fetch(`${POST_API_BASE_URL}/api/analytics`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });

        if (!response.ok) {
          throw new Error('Failed to fetch analytics.');
        }

        const data = await response.json();
        setAnalyticsData(data);
      } catch (e) {
        console.error('Fetch error:', e);
      }
    };

    if (token) {
      fetchAccounts();
      fetchAnalytics();
    }
  }, [token]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="animate-spin text-blue-600 w-12 h-12" />
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="p-10 text-center text-red-500 font-bold">
        <p>Error: {error}</p>
        <p>Please try logging in again.</p>
      </div>
    );
  }

  return (
    <div className="p-6 md:p-10 space-y-8">
      <h1 className="text-4xl font-bold text-gray-900 mb-6">Dashboard</h1>
      
      {/* Account Overview Section */}
      <div className="bg-white p-6 rounded-xl shadow-lg">
        <h2 className="text-2xl font-semibold text-gray-800 mb-6">Connected Accounts</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {accounts.length > 0 ? (
            accounts.map(account => (
              <AccountCard key={account.platformUserId} account={account} />
            ))
          ) : (
            <p className="text-gray-500 text-center col-span-3">No accounts connected yet. Please go to your profile to connect a new one.</p>
          )}
        </div>
      </div>
      
      {/* Quick Analytics Section */}
      <div className="bg-white p-6 rounded-xl shadow-lg">
        <h2 className="text-2xl font-semibold text-gray-800 mb-6">Engagement Over Time</h2>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={analyticsData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="name" stroke="#6b7280" />
            <YAxis stroke="#6b7280" />
            <Tooltip />
            <Line type="monotone" dataKey="Meta" stroke="#4c51bf" strokeWidth={2} activeDot={{ r: 8 }} />
            <Line type="monotone" dataKey="TikTok" stroke="#06b6d4" strokeWidth={2} activeDot={{ r: 8 }} />
            <Line type="monotone" dataKey="Snapchat" stroke="#ef4444" strokeWidth={2} activeDot={{ r: 8 }} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

// Login component with OAuth buttons.
const Login = () => {
  const handleLogin = (platform) => {
    window.location.href = `${AUTH_API_BASE_URL}/oauth/${platform}/login`;
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100 p-6">
      <div className="bg-white p-10 rounded-xl shadow-2xl text-center max-w-md w-full space-y-6">
        <h1 className="text-4xl font-extrabold text-gray-900">Welcome</h1>
        <p className="text-gray-600">Please connect your social media accounts to get started.</p>
        <button
          onClick={() => handleLogin('meta')}
          className="w-full py-3 px-4 rounded-lg text-white font-semibold transition-colors bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 flex items-center justify-center"
        >
          <img src="https://upload.wikimedia.org/wikipedia/commons/thumb/b/b8/2021_Facebook_icon.svg/1024px-2021_Facebook_icon.svg.png" alt="Meta Icon" className="h-6 w-6 mr-2" />
          Connect with Meta
        </button>
        <button
          onClick={() => handleLogin('tiktok')}
          className="w-full py-3 px-4 rounded-lg text-white font-semibold transition-colors bg-black hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-gray-700 flex items-center justify-center"
        >
          <img src="https://www.tiktok.com/favicon.ico" alt="TikTok Icon" className="h-6 w-6 mr-2" />
          Connect with TikTok
        </button>
        <button
          onClick={() => handleLogin('snapchat')}
          className="w-full py-3 px-4 rounded-lg text-gray-800 font-semibold transition-colors bg-yellow-400 hover:bg-yellow-500 focus:outline-none focus:ring-2 focus:ring-yellow-300 flex items-center justify-center"
        >
          <img src="https://www.snapchat.com/favicon.ico" alt="Snapchat Icon" className="h-6 w-6 mr-2" />
          Connect with Snapchat
        </button>
      </div>
    </div>
  );
};

// Component to handle the redirect from the OAuth callback.
const AuthSuccess = ({ setToken }) => {
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const jwtToken = urlParams.get('token');
    
    if (jwtToken) {
      localStorage.setItem('jwtToken', jwtToken);
      setToken(jwtToken);
      window.location.href = '/';
    } else {
      console.error('JWT token not found in URL.');
      window.location.href = '/login';
    }
  }, [setToken]);

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <div className="text-center p-10">
        <h1 className="text-3xl font-bold">Authentication Successful!</h1>
        <p className="mt-4 text-gray-600">Redirecting to your dashboard...</p>
        <Loader2 className="animate-spin text-blue-600 w-12 h-12 mx-auto mt-6" />
      </div>
    </div>
  );
};

// New Post Creator component.
const PostCreator = ({ token, onPostCreated }) => {
  const [content, setContent] = useState('');
  const [platform, setPlatform] = useState('');
  const [scheduledAt, setScheduledAt] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [accounts, setAccounts] = useState([]);

  useEffect(() => {
    const fetchAccounts = async () => {
      if (!token) return;
      try {
        const response = await fetch(`${ACCOUNT_API_BASE_URL}/api/accounts`, {
          headers: { 'Authorization': `Bearer ${token}` },
        });
        const data = await response.json();
        setAccounts(data);
        if (data.length > 0) {
          setPlatform(data[0].platform);
        }
      } catch (error) {
        console.error('Failed to fetch accounts:', error);
      }
    };
    fetchAccounts();
  }, [token]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const response = await fetch(`${POST_API_BASE_URL}/api/posts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          platform: platform,
          content: content,
          scheduledAt: new Date(scheduledAt).toISOString(),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to create post.');
      }

      const newPost = await response.json();
      console.log('Post created:', newPost);
      alert('Post scheduled successfully!');
      if (onPostCreated) {
        onPostCreated();
      }
    } catch (error) {
      console.error('Error creating post:', error);
      alert('Failed to schedule post.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="p-6 md:p-10">
      <h1 className="text-4xl font-bold text-gray-900 mb-6">Create New Post</h1>
      <div className="bg-white p-6 rounded-xl shadow-lg max-w-2xl mx-auto">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-gray-700 font-semibold mb-2">Platform</label>
            <select
              value={platform}
              onChange={(e) => setPlatform(e.target.value)}
              className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              required
            >
              <option value="">Select a platform</option>
              {accounts.map(acc => (
                <option key={acc.platform} value={acc.platform}>{acc.platform} ({acc.username})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-gray-700 font-semibold mb-2">Content</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              rows="5"
              placeholder="What would you like to post?"
              required
            ></textarea>
          </div>
          <div>
            <label className="block text-gray-700 font-semibold mb-2">Schedule Time</label>
            <input
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              required
            />
          </div>
          <button
            type="submit"
            className="w-full py-3 px-4 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors flex items-center justify-center disabled:opacity-50"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="animate-spin mr-2 h-5 w-5" />
                Scheduling...
              </>
            ) : (
              'Schedule Post'
            )}
          </button>
        </form>
      </div>
    </div>
  );
};

// Scheduler component (updated to fetch data).
const Scheduler = ({ token }) => {
  const [posts, setPosts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchPosts = async () => {
      try {
        const response = await fetch(`${POST_API_BASE_URL}/api/posts`, {
          headers: { 'Authorization': `Bearer ${token}` },
        });
        const data = await response.json();
        setPosts(data);
      } catch (error) {
        console.error('Failed to fetch posts:', error);
      } finally {
        setIsLoading(false);
      }
    };

    if (token) {
      fetchPosts();
    }
  }, [token]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="animate-spin text-blue-600 w-12 h-12" />
      </div>
    );
  }

  return (
    <div className="p-6 md:p-10">
      <h1 className="text-4xl font-bold text-gray-900 mb-6">Post Scheduler</h1>
      <div className="bg-white p-6 rounded-xl shadow-lg">
        <h2 className="text-2xl font-semibold text-gray-800 mb-4">Upcoming Posts</h2>
        <div className="space-y-4">
          {posts.length > 0 ? (
            posts.map(post => (
              <div key={post.id} className="p-4 bg-gray-50 rounded-lg border border-gray-200 flex items-center">
                <span className="text-gray-500 mr-4">
                  {post.platform === 'Meta' ? '🌐' : post.platform === 'TikTok' ? '🎵' : '👻'}
                </span>
                <div className="flex-grow">
                  <p className="font-medium text-gray-700">{post.content}</p>
                  <p className="text-sm text-gray-400">{new Date(post.scheduledAt).toLocaleString()}</p>
                </div>
                <button className="text-sm text-blue-600 hover:text-blue-800 transition-colors">Edit</button>
              </div>
            ))
          ) : (
            <p className="text-gray-500 text-center">No posts scheduled yet. Create one now!</p>
          )}
        </div>
      </div>
    </div>
  );
};

// Analytics component (updated to fetch data).
const Analytics = ({ token }) => {
  const [analyticsData, setAnalyticsData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchAnalytics = async () => {
      try {
        const response = await fetch(`${POST_API_BASE_URL}/api/analytics`, {
          headers: { 'Authorization': `Bearer ${token}` },
        });
        const data = await response.json();
        setAnalyticsData(data);
      } catch (error) {
        console.error('Failed to fetch analytics:', error);
      } finally {
        setIsLoading(false);
      }
    };
    
    if (token) {
      fetchAnalytics();
    }
  }, [token]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="animate-spin text-blue-600 w-12 h-12" />
      </div>
    );
  }

  return (
    <div className="p-6 md:p-10">
      <h1 className="text-4xl font-bold text-gray-900 mb-6">Analytics & Reporting</h1>
      <div className="bg-white p-6 rounded-xl shadow-lg">
        <h2 className="text-2xl font-semibold text-gray-800 mb-6">Engagement Over Time</h2>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={analyticsData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="name" stroke="#6b7280" />
            <YAxis stroke="#6b7280" />
            <Tooltip />
            <Line type="monotone" dataKey="Meta" stroke="#4c51bf" strokeWidth={2} activeDot={{ r: 8 }} />
            <Line type="monotone" dataKey="TikTok" stroke="#06b6d4" strokeWidth={2} activeDot={{ r: 8 }} />
            <Line type="monotone" dataKey="Snapchat" stroke="#ef4444" strokeWidth={2} activeDot={{ r: 8 }} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

// Engagement component (unified inbox).
const Engagement = () => {
  return (
    <div className="p-6 md:p-10">
      <h1 className="text-4xl font-bold text-gray-900 mb-6">Engagement Management</h1>
      <div className="bg-white p-6 rounded-xl shadow-lg">
        <h2 className="text-2xl font-semibold text-gray-800 mb-4">Unified Inbox</h2>
        <p className="text-gray-600">This area would function as a unified inbox, fetching comments and direct messages from all connected social media accounts. Users can view and respond to interactions from a single, centralized location, with notifications for new activity.</p>
        <div className="mt-4 p-4 bg-gray-50 rounded-lg">
          <p className="font-medium text-gray-700">Mock Comment from TikTok:</p>
          <p className="text-sm text-gray-500">"Love this video! 😍"</p>
        </div>
        <div className="mt-2 p-4 bg-gray-50 rounded-lg">
          <p className="font-medium text-gray-700">Mock Message from Meta:</p>
          <p className="text-sm text-gray-500">"Hi, is this product available?"</p>
        </div>
      </div>
    </div>
  );
};

const App = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [token, setToken] = useState(localStorage.getItem('jwtToken'));
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const renderContent = () => {
    if (!token) {
      if (window.location.pathname === '/auth-success') {
        return <AuthSuccess setToken={setToken} />;
      }
      return <Login />;
    }

    switch (currentPage) {
      case 'dashboard':
        return <Dashboard token={token} />;
      case 'scheduler':
        return <Scheduler token={token} />;
      case 'analytics':
        return <Analytics token={token} />;
      case 'engagement':
        return <Engagement />;
      case 'new-post':
        return <PostCreator token={token} onPostCreated={() => setCurrentPage('scheduler')} />;
      default:
        return <Dashboard token={token} />;
    }
  };

  return (
    <div className="min-h-screen bg-gray-100 font-sans text-gray-800 flex flex-col lg:flex-row">
      {/* Mobile Header */}
      <header className="lg:hidden bg-white shadow-md p-4 flex items-center justify-between sticky top-0 z-50">
        <h1 className="text-2xl font-bold text-blue-600">SMM Platform</h1>
        <button onClick={() => setIsSidebarOpen(!isSidebarOpen)} className="p-2 rounded-md hover:bg-gray-200 transition-colors">
          <Menu className="h-6 w-6 text-gray-600" />
        </button>
      </header>

      {token && (
        <aside className={`bg-gray-900 text-white w-64 p-6 flex flex-col transition-transform duration-300 ease-in-out ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full'} lg:translate-x-0 fixed inset-y-0 left-0 z-40 lg:sticky lg:top-0 lg:h-screen shadow-2xl`}>
          <div className="flex-shrink-0 flex items-center justify-between mb-8">
            <h1 className="text-3xl font-extrabold text-white">SMM Tool</h1>
            <button onClick={() => setIsSidebarOpen(false)} className="lg:hidden p-2 rounded-full hover:bg-gray-800 transition-colors">
              <Menu className="h-6 w-6 text-gray-400" />
            </button>
          </div>
          <nav className="flex-grow">
            <ul className="space-y-2">
              <li>
                <button
                  onClick={() => { setCurrentPage('dashboard'); setIsSidebarOpen(false); }}
                  className={`flex items-center w-full p-3 rounded-lg transition-colors duration-200 ${currentPage === 'dashboard' ? 'bg-blue-600 text-white' : 'hover:bg-gray-800 text-gray-300'}`}
                >
                  <LayoutDashboard className="mr-3 h-5 w-5" />
                  <span>Dashboard</span>
                </button>
              </li>
              <li>
                <button
                  onClick={() => { setCurrentPage('scheduler'); setIsSidebarOpen(false); }}
                  className={`flex items-center w-full p-3 rounded-lg transition-colors duration-200 ${currentPage === 'scheduler' ? 'bg-blue-600 text-white' : 'hover:bg-gray-800 text-gray-300'}`}
                >
                  <Calendar className="mr-3 h-5 w-5" />
                  <span>Scheduler</span>
                </button>
              </li>
              <li>
                <button
                  onClick={() => { setCurrentPage('analytics'); setIsSidebarOpen(false); }}
                  className={`flex items-center w-full p-3 rounded-lg transition-colors duration-200 ${currentPage === 'analytics' ? 'bg-blue-600 text-white' : 'hover:bg-gray-800 text-gray-300'}`}
                >
                  <BarChart2 className="mr-3 h-5 w-5" />
                  <span>Analytics</span>
                </button>
              </li>
              <li>
                <button
                  onClick={() => { setCurrentPage('engagement'); setIsSidebarOpen(false); }}
                  className={`flex items-center w-full p-3 rounded-lg transition-colors duration-200 ${currentPage === 'engagement' ? 'bg-blue-600 text-white' : 'hover:bg-gray-800 text-gray-300'}`}
                >
                  <MessageSquare className="mr-3 h-5 w-5" />
                  <span>Engagement</span>
                </button>
              </li>
            </ul>
          </nav>
          <div className="mt-auto">
            <button
              onClick={() => { setCurrentPage('new-post'); setIsSidebarOpen(false); }}
              className="flex items-center w-full p-3 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors"
            >
              <Plus className="mr-2 h-5 w-5" />
              <span>New Post</span>
            </button>
          </div>
        </aside>
      )}

      {/* Main Content Area */}
      <main className="flex-grow p-4 lg:p-8 overflow-y-auto">
        {renderContent()}
      </main>
    </div>
  );
};

export default App;
