.. _reference-tasks-transfer:

########
TRANSFER
########

Le traitement ``TRANSFER`` programme un nouveau transfert sur la même instance
de Gateway avec une date de démarrage immédiate. Les arguments sont:

* ``file`` (*string*) - Le chemin du fichier à transférer.
* ``using`` (*string*) - Le client à utiliser pour le transfert.
* ``to`` (*string*) - Le nom du partenaire auquel se connecter.
* ``as`` (*string*) - Le nom du compte avec lequel s'authentifier auprès du partenaire.
* ``rule`` (*string*) - Le nom de la règle à utiliser pour le transfert.
* ``copyInfo`` (*boolean*) - Indique si les informations du transfert en cours
  doit être copiées sur le nouveau transfert programmé.
* ``info`` (*object*) - Les informations de transfert du nouveau transfert. Si
  les informations du transfert en cours ont été copiées sur le nouveau transfert
  (via le paramètre **copyInfo** décris ci-dessus), les nouvelles informations
  indiquées ici viendront s'additionner à celles-ci (et les écraseront en cas
  de conflit).
