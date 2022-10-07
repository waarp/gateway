.. _reference-cmd-waarp-gatewayd-purge:

########################
``waarp-gatewayd purge``
########################

.. program:: waarp-gatewayd purge

``waarp-gatewayd purge`` est la commande permettant de purger l'historique de
transfert afin de libérer de l'espace disque sur la base de données.

.. warning:: Évidemment, cette purge est irréversible, soyez donc prudent lorsque
   vous l'utilisez. La commande vous demandera confirmation avant d'effectuer la
   purge.

Cette commande accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --reset, -r

   Si cette option est présente, en plus de purger d'historique, la commande
   réinitialisera l'auto-incrément des identifiants locaux de transfert à zéro.

   .. warning:: Cette option ne peut être utilisée que si la table des transferts
      en cours est vide. Dans le cas contraire, la commande échouera.

.. option:: --verbose, -v

   Active l'écriture des logs sur la sortie d'erreur.
   Cet argument peut être répété jusqu'à 3 fois pour augmenter la verbosité
   (ex : ``-vvv``).