import ReactDOM from 'react-dom'

import 'normalize.css/normalize.css'
import '@blueprintjs/icons/lib/css/blueprint-icons.css'
import '@blueprintjs/core/lib/css/blueprint.css'
import './index.css'

import { FocusStyleManager } from '@blueprintjs/core'
import App from './App'
import * as serviceWorker from './serviceWorker'

FocusStyleManager.onlyShowFocusOnTabs()
ReactDOM.render(<App />, document.getElementById('root'))

serviceWorker.register()
