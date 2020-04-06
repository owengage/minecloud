import React, {useState, useEffect} from 'react'
import ReactDOM from 'react-dom'

import Amplify, { Auth } from 'aws-amplify';
import { withAuthenticator } from 'aws-amplify-react'

import {
  BrowserRouter as Router,
  Switch,
  Route,
  Link
} from "react-router-dom";

import ProtoPage from './proto-page'
import awsconfig from './aws-exports'

Amplify.configure(awsconfig);

function App() {
  const [user, setUser] = useState(null);

  useEffect(() => {
      (async () => {
          const user = await Auth.currentAuthenticatedUser();
          setUser(user);
      })()
  }, []);

  if (user) {
    return <ProtoPage user={user} />
  }
  return "Not signed in";
}

Auth.currentSession().then(console.log).catch(console.error)

const AuthedApp = withAuthenticator(App)
const wrapper = document.getElementById("app");
wrapper ? ReactDOM.render(<AuthedApp />, wrapper) : false;