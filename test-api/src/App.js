import React, { useState } from 'react';
import './App.css';
import HomePage from './components/HomePage';
import JwtRpcApp from './components/JwtRpcApp';
import CacheMetrics from './components/CacheMetrics';

function App() {
  const [currentApp, setCurrentApp] = useState('home');

  const navigateToApp = (appId) => {
    setCurrentApp(appId);
  };

  const navigateToHome = () => {
    setCurrentApp('home');
  };

  const renderCurrentApp = () => {
    switch (currentApp) {
      case 'jwt-rpc':
        return <JwtRpcApp onBackToHome={navigateToHome} />;
      case 'cache-metrics':
        return <CacheMetrics onBackToHome={navigateToHome} />;
      case 'home':
      default:
        return <HomePage onNavigateToApp={navigateToApp} />;
    }
  };

  return renderCurrentApp();
}

export default App; 