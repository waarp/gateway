##################################
Purge de l'historique de transfert
##################################

Afin d'éviter que la base de données ne devienne trop large (en particulier lorsque
la fréquence de transfert sur la *gateway* est élevée), il est parfois nécessaire
de purger l'historique de transfert afin de libérer de l'espace disque.

Pour ce faire, l'exécutable ``waarp-gatewayd`` inclus une commande ``purge``
permettant de vider la table d'historique. Cette commande inclue les options
suivantes :

- ``-c``: Le fichier de configuration de la *gateway* (contient les informations
  de connexion à la base de données). *REQUIS*
- ``-r``: Réinitialise l'auto-incrément des identifiants de transfert. Par défaut,
  la commande ``purge`` ne réinitialise pas cet auto-incrément car cela requiert
  que la table des transferts en cours soit vide (afin d'éviter d'éventuels conflits).
  Une fois l'auto-incrément réinitialisé, les identifiants des prochains transferts
  reprendront à zéro.

  .. note:: Nous parlons ici des identifiants de transferts **locaux** (*id*),
     et non des identifiants publics (*remoteTransferID*), ces derniers n'étant
     pas générés par un auto-incrément.