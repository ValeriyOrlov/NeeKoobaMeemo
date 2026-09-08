import './auth.js';
import './lobby.js';
import { loadProfile } from './api.js';
import { showScreen } from './ui.js';

window.onload = () => {
  if (localStorage.getItem('game_token')) {
    loadProfile();
  } else {
    showScreen('auth');
  }
};