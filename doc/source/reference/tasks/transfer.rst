.. _reference-tasks-transfer:

########
TRANSFER
########

Le traitement ``TRANSFER`` programme un nouveau transfert sur la même instance
de Gateway avec une date de démarrage immédiate. Les arguments sont:

* ``file`` (*string*) - Le chemin du fichier à transférer.
* ``output`` (*string*) - Le chemin destination du fichier transféré. Par défaut,
  le nom d'origine du fichier est utilisé.
* ``using`` (*string*) - Le client à utiliser pour le transfert.
* ``to`` (*string*) - Le nom du partenaire auquel se connecter.
* ``as`` (*string*) - Le nom du compte avec lequel s'authentifier auprès du partenaire.
* ``rule`` (*string*) - Le nom de la règle à utiliser pour le transfert.
* ``using`` (*string*) - Le nom du client à utiliser pour faire le transfert.
  Si omit, un client par défaut sera utilisé (si possible).
* ``copyInfo`` (*boolean*) - Indique si les informations du transfert en cours
  doit être copiées sur le nouveau transfert programmé.
* ``info`` (*object*) - Les informations de transfert du nouveau transfert. Si
  les informations du transfert en cours ont été copiées sur le nouveau transfert
  (via le paramètre **copyInfo** décris ci-dessus), les nouvelles informations
  indiquées ici viendront s'additionner à celles-ci (et les écraseront en cas
  de conflit).
* ``nbOfAttempts`` (*number*) - Le nombre de fois que le transfert sera automatiquement
  re-tenté en cas d'échec.
* ``firstRetryDelay`` (*number*) - Le délai entre le transfert original et la première
  reprise automatique. Les unités acceptées sont ``h`` (heures), ``m`` (minutes) et
  ``s`` (secondes), par exemple "1h30m15s". Ne peut être inférieur à 1s.
* ``retryIncrementFactor```(*number*) - Le facteur par lequel le délai ci-dessus sera
  multiplié à chaque nouvelle tentative. Les nombres décimaux sont acceptés.
