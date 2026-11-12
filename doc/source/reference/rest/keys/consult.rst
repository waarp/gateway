Consulter une clé cryptographique
=================================

.. http:get:: /api/keys/(string:key_name)

   Renvoie la clé cryptographique demandée. La valeur d'une clé secrète (AES,
   HMAC, PGP privée) n'est jamais renvoyée : seule celle d'une clé PGP publique
   l'est.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La clé a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La clé demandée n'existe pas

   :resjson string name: Le nom de la clé cryptographique.
   :resjson string type: Le type de la clé cryptographique. Les valeurs possibles
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :resjson string key: [``PGP-PUBLIC`` uniquement] La clé publique, en format
      textuel. Absent pour les clés secrètes (``AES``, ``HMAC``,
      ``PGP-PRIVATE``), dont la valeur n'est jamais renvoyée.
   :resjson string fingerprint: [Clés PGP uniquement] L'empreinte de la clé, en
      hexadécimal.
   :resjson array identities: [Clés PGP uniquement] Les identités portées par la
      clé.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/keys/pgp-key HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 163

      {
        "name": "pgp-key",
        "type": "PGP-PRIVATE",
        "fingerprint": "57e09fbb394eec758040cdc4b9f43dcb4c634a87",
        "identities": ["Waarp (Waarp PGP test key) <info@waarp.org>"]
      }
