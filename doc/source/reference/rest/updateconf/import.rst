.. _reference-rest-updateconf-import:

Importer une configuration
===========================

.. http:post:: /api/updateconf

   Importe tout ou partie d'un fichier de configuration au format JSON ou YAML,
   donné dans le corps de la requête.

   La structure et le contenu attendus du fichier sont documentés :any:`ici
   <reference-backup-json>`. Il s'agit du même format que celui utilisé par les
   commandes :any:`reference-cmd-waarp-gatewayd-import` et
   :any:`reference-cmd-waarp-gatewayd-export`, ainsi que par le point d'accès
   :any:`reference-rest-updateconf-export`.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqheader Content-Type: Le format du corps de la requête. Les valeurs
      acceptées sont ``application/json`` (par défaut) et ``application/yaml``.

   :reqheader Targets: [Optionnel, défaut : ``all``] Restreint l'import à une
      ou plusieurs catégories de données. Cet en-tête peut être renseigné
      plusieurs fois afin de spécifier plusieurs catégories.

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

   :reqheader Reset: [Optionnel, défaut : ``false``] Si renseigné à une valeur
      *vraie* (``true``, ``yes`` ou ``1``), la (ou les) catégorie(s) de données
      ciblée(s) par l'import sera(seront) intégralement vidée(s) avant que les
      nouvelles données ne soient importées.

   :reqheader Dry-Run: [Optionnel, défaut : ``false``] Si renseigné à une
      valeur *vraie* (``true``, ``yes`` ou ``1``), l'import est simulé, mais
      les changements sont annulés à la fin de l'opération. Utile pour vérifier
      que le fichier donné est valide sans en appliquer réellement le contenu.

   :reqheader Restart: [Optionnel, défaut : ``false``] Si renseigné à une
      valeur *vraie* (``true``, ``yes`` ou ``1``), les services concernés par
      les éléments importés (:term:`serveurs locaux<serveur>`, clients et
      *filewatchers*) seront redémarrés une fois l'import terminé.

   :statuscode 201: La configuration a été importée avec succès
   :statuscode 400: Le contenu de la requête est invalide
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 403: L'utilisateur n'a pas le droit d'effectuer cette action

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/updateconf HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Targets: rules
         Reset: true
         Restart: true
         Content-Length: 96

         {
           "rules": [
             {
               "name": "my_rule",
               "isSend": false,
               "path": "/rule_path"
             }
           ]
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
