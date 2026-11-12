Lister les clés cryptographiques
================================

.. http:get:: /api/keys

   Renvoie une liste des clés cryptographiques remplissant les critères donnés en
   paramètres de requête. La valeur d'une clé secrète (AES, HMAC, PGP privée)
   n'est jamais renvoyée : seule celle d'une clé PGP publique l'est.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les clés seront triées *(défaut: name+)*.
      Valeurs autorisées : ``name+``, ``name-``, ``type+``, ``type-``.
   :type sort: string

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array cryptoKeys: La liste des clés demandées
   :resjsonarr string name: Le nom de la clé cryptographique.
   :resjsonarr string type: Le type de la clé cryptographique. Les valeurs possibles
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :resjsonarr string key: [``PGP-PUBLIC`` uniquement] La clé publique, en format
      textuel. Absent pour les clés secrètes (``AES``, ``HMAC``,
      ``PGP-PRIVATE``), dont la valeur n'est jamais renvoyée.
   :resjsonarr string fingerprint: [Clés PGP uniquement] L'empreinte de la clé,
      en hexadécimal.
   :resjsonarr array identities: [Clés PGP uniquement] Les identités portées par
      la clé.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/keys?limit=10&sort=name- HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 318

      {
        "cryptoKeys": [{
          "name": "pgp-public-key",
          "type": "PGP-PUBLIC",
          "key": "-----BEGIN PGP PUBLIC KEY BLOCK----- [...]",
          "fingerprint": "57e09fbb394eec758040cdc4b9f43dcb4c634a87",
          "identities": ["Waarp (Waarp PGP test key) <info@waarp.org>"]
        },{
          "name": "hmac-key",
          "type": "HMAC"
        },{
          "name": "aes-key",
          "type": "AES"
        }]
      }
