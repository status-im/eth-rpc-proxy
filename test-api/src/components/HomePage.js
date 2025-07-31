import React from 'react';

function HomePage({ onNavigateToApp }) {
  const apps = [
    {
      id: 'jwt-rpc',
      title: '🔐 JWT RPC Test App',
      description: 'Test JWT authentication and RPC requests to the proxy',
      status: 'Available'
    },
    {
      id: 'cache-metrics',
      title: '📊 Cache Metrics Dashboard',
      description: 'Real-time cache performance and usage metrics monitoring',
      status: 'Available'
    },
    // Other utilities can be added here
  ];

  return (
    <div className="App">
      <header className="App-header">
        <h1>🛠️ Test Utilities Dashboard</h1>
        <p>Collection of testing and development utilities</p>
      </header>
      
      <main className="App-main">
        <div className="home-container">
          <div className="apps-grid">
            {apps.map((app) => (
              <div key={app.id} className="app-card">
                <h3>{app.title}</h3>
                <p>{app.description}</p>
                <div className="app-status">
                  <span className="status-badge success">{app.status}</span>
                </div>
                <button 
                  className="button app-button"
                  onClick={() => onNavigateToApp(app.id)}
                >
                  Open App →
                </button>
              </div>
            ))}
            
            {/* Placeholder for future utilities */}
            <div className="app-card placeholder">
              <h3>➕ More Tools Coming Soon</h3>
              <p>Additional utilities will be added here</p>
              <div className="app-status">
                <span className="status-badge">Coming Soon</span>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

export default HomePage;