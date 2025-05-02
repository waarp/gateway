.. _reference-cmd-waarp-gatewayd-import:

#########################
``waarp-gatewayd import``
#########################


.. program:: waarp-gatewayd import

``waarp-gatewayd import`` est une commande qui permet de charger la
configuration de la Gateway depuis un fichier JSON.

La structure et le contenu du fichier JSON est documenté :any:`ici
<reference-backup-json>`. Il peut également être généré avec la commande
:any:`reference-cmd-waarp-gatewayd-export`.

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

.. option:: --source FILE, -s FILE

   :Défaut: entrée standard

   Indique le chemin vers le fichier contenant les données à importer. Peut être
   un fichier JSON ou YAML.

.. option:: -t [rules|servers|partners|clients|users|clouds|snmp|authorities|keys|all], --target [rules|servers|partners|clients|users|clouds|snmp|authorities|keys|all]

   :Défaut: ``all``

   Spécifie un sous-ensemble de données à importer. Cet argument peut être
   renseigné plusieurs fois pour choisir plusieurs catégories.

   Les valeurs possibles sont :

   * ``rules``: Règles de transfert.
   * ``servers``: Définitions de serveurs locaux, incluant les comptes locaux et
     certificats associés.
   * ``partners``: Définitions de partenaires distants, incluant les comptes
     locaux et certificats associés.
   * ``users``: Identifiants des utilisateurs Waarp Gateway servant à l'administration.
   * ``all``: Toutes les données contenues dans le fichier.

.. option:: --reset-before-import, -r

   Réinitialise la base de données avant l'import. Les tables concernées par
   l'import seront vidées avant l'import. Un message de confirmation s'affichera
   avant d'effectuer la réinitialisation.

   ATTENTION: Cette option est incompatible avec l'import de données depuis la
   sortie standard. Il vous faudra soit importer les données depuis un fichier,
   soit utiliser l'option :option:`--force-reset-before-import` décrite ci-dessous.

.. option:: --force-reset-before-import

   Similaire à ``--reset-before-import, -r`` ci-dessus mais sans le message de
   confirmation. Utile pour les scripts.

.. option:: --dry-run, -d

   Simule l'import sans modifier aucune donnée.

.. option:: --verbose, -v

   Active l'écriture des logs sur la sortie d'erreur.
   Cet argument peut être répété jusqu'à 3 fois pour augmenter la verbosité
   (ex : ``-vvv``).

.. option:: --help, -h

   Affiche l'aide de la commande.
