.. _ref-tasks-preregister:

###########
PREREGISTER
###########

.. note:: À ne pas confondre avec la tâche *TRANSFER*. Celle-ci permet de lancer
   des transferts **client** qui s'exécutent immédiatement à l'initiative de
   la Gateway.

Le traitement ``PREREGISTER`` pré-enregistre un nouveau transfert serveur. Ce
transfert restera en attente jusqu'à ce que le partenaire renseigné vienne le
récupérer, ou bien jusqu'à ce que le transfert expire. Les arguments de la tâche
sont :

* ``file`` (*string*) - Le chemin du fichier à transférer.
* ``rule`` (*string*) - Le nom de la règle à utiliser pour le transfert.
* ``isSend`` (*boolean*) - Indique si le transfert sera un envoi (*true*) ou une
  réception (*false*).
* ``server`` (*string*) - Le nom du serveur local sur lequel la requête de
  transfert devra être reçue.
* ``account`` (*string*) - Le login du partenaire qui sera utilisé par le partenaire
  pour faire la requête de transfert.
* ``dueDate`` (*string*) - La date limite (en format ISO-8601) du transfert. Au
  delà de cette date, le transfert expirera et tombera en erreur.
* ``copyInfo`` (*boolean*) - Indique si les informations du transfert en cours
  doit être copiées sur le nouveau transfert pré-enregistré.
* ``info`` (*object*) - Les informations de transfert du nouveau transfert. Si
  les informations du transfert en cours ont été copiées sur le nouveau transfert
  (via le paramètre **copyInfo** décris ci-dessus), les nouvelles informations
  indiquées ici viendront s'additionner à celles-ci (et les écraseront en cas
  de conflit).
