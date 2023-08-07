##################################
Purge de l'historique de transfert
##################################

Afin d'éviter que la base de données ne devienne trop large (en particulier lorsque
la fréquence de transfert sur Waarp Gateway est élevée), il est parfois nécessaire
de purger l'historique de transfert afin de libérer de l'espace disque.

Pour ce faire, l'exécutable ``waarp-gatewayd`` inclus une commande ``purge``
permettant de vider la table d'historique. Cette commande inclue les options
suivantes :

- ``-c``: Le fichier de configuration de Waarp Gateway (contient les informations
  de connexion à la base de données). *REQUIS*
- ``-o``: Limite la purge aux transferts plus anciens que le temps donné. Peut
  être soit une date soit une durée (voir la :ref:`documentation
  <reference-cmd-waarp-gatewayd-purge-older-than>` de la commande pour plus
  de détails).
- ``-r``: Réinitialise l'auto-incrément des identifiants de transfert. Par défaut,
  la commande ``purge`` ne réinitialise pas cet auto-incrément car cela requiert
  que la table des transferts en cours soit vide (afin d'éviter d'éventuels conflits).
  Une fois l'auto-incrément réinitialisé, les identifiants des prochains transferts
  reprendront à zéro. Cette option est également incompatible avec l'option ``-o``
  décrite ci-dessus. Les 2 options ne peuvent donc pas être utilisée en même temps.

  .. note::
     Nous parlons ici des identifiants de transferts **locaux** (``id``), et non des
     identifiants publics (``remoteTransferID``), ces derniers n'étant pas générés
     par un auto-incrément.
