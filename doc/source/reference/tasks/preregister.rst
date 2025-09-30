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
* ``validFor`` (*string*) - La durée pour laquelle le transfert est valide. Au delà,
  le transfert tombera en erreur. Les unités de temps acceptées sont ``d`` (jours),
  ``h`` (heures), ``m`` (minutes) et ``s`` (secondes).
* ``copyInfo`` (*boolean*) - Indique si les informations du transfert en cours
  doivent être copiées sur le nouveau transfert pré-enregistré.
* ``info`` (*object*) - Les informations de transfert du nouveau transfert. Si
  les informations du transfert en cours ont été copiées sur le nouveau transfert
  (via le paramètre **copyInfo** décris ci-dessus), les nouvelles informations
  indiquées ici viendront s'additionner à celles-ci (et les écraseront en cas
  de conflit).
