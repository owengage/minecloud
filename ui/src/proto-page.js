import React, { useEffect, useState } from 'react';
import { Auth } from 'aws-amplify';

function ProtoPage({user}) {
    const handleSignOut = () => {
        Auth.signOut()
    }
    return (
        <div className="proto">
            <header>
                <h1>Minecloud</h1>
                <nav>
                    <ul>
                        <li>
                            <a onClick={handleSignOut} href='#'>Sign out</a>
                        </li>
                    </ul>
                </nav>
            </header>
            <article>
                
            </article>
        </div>
    );
}

export default ProtoPage