===========================
Importer une configuration
===========================

.. program:: waarp-gateway updateconf import

Importe tout ou partie d'un fichier de configuration, au format JSON ou YAML,
vers la Gateway.

La structure et le contenu attendus du fichier sont documentés :any:`ici
<reference-backup-json>`. Il s'agit du même format que celui utilisé par les
commandes :any:`reference-cmd-waarp-gatewayd-import` et
:any:`reference-cmd-waarp-gatewayd-export`, ainsi que par le point d'accès
:any:`reference-rest-updateconf-export`.

**Commande**

.. code-block:: shell

   waarp-gateway updateconf import --source "<FILE>"

**Options**

.. option:: -s <FILE>, --source=<FILE>

   Le fichier de configuration à importer, au format JSON ou YAML.

.. option:: -t [rules|servers|partners|clients|users|clouds|snmp|authorities|keys|email|filewatchers|all], --target=[rules|servers|partners|clients|users|clouds|snmp|authorities|keys|email|filewatchers|all]

   :Défaut: ``all``

   Restreint l'import à un sous-ensemble de données. Cette option peut être
   renseignée plusieurs fois pour importer plusieurs sous-ensembles à la fois.

   Les valeurs possibles sont :

   * ``rules``: Règles de transfert.
   * ``servers``: Définitions de serveurs locaux, incluant les comptes locaux
     et informations d'authentification associées.
   * ``partners``: Définitions de partenaires distants, incluant les comptes
     locaux et informations d'authentification associées.
   * ``clients``: Définitions de clients de transfert.
   * ``users``: Identifiants des utilisateurs Waarp Gateway servant à
     l'administration.
   * ``clouds``: Instances de stockage dans le *cloud*.
   * ``snmp``: Configuration du service SNMP (serveur et *monitors*).
   * ``authorities``: Autorités d'authentification.
   * ``keys``: Clés cryptographiques.
   * ``email``: Modèles et informations d'authentification d'envoi d'e-mails.
   * ``filewatchers``: *Filewatchers*.
   * ``all``: Toutes les catégories de données présentes dans le fichier.

.. option:: -d, --dry-run

   Simule l'import sans effectuer aucun changement. Utile pour vérifier que le
   fichier donné est valide sans en appliquer réellement le contenu.

.. option:: -r, --reset

   Vide entièrement la (ou les) catégorie(s) de données ciblée(s) avant d'y
   importer les éléments contenus dans le fichier.

.. option:: --restart

   Redémarre les services concernés par les éléments importés
   (:term:`serveurs locaux<serveur>`, clients et *filewatchers*) une fois
   l'import terminé.

|

**Exemple**

.. code-block:: shell

   waarp-gateway updateconf import --source 'config.json' --target 'rules' --reset --restart
