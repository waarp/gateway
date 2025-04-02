Lister les clés cryptographiques
================================

.. http:get:: /api/keys

   Renvoie une liste des clés cryptographiques remplissant les critères donnés en
   paramètres de requête.

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

   :resjson array cryptoKeys: La liste des clés demandés
   :resjsonarr string name: Le nom de la clé cryptographique.
   :resjsonarr string type: Le type de la clé cryptographique. Les valeurs possibles
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :resjsonarr string key: La représentation textuelle de la clé. Si la clé n'est
      pas nativement en format textuel, celle-ci sera convertie en Base64.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/keys?limit=10&sort=name- HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 76

      {
        "cryptoKeys": [{
          "name": "aes-key",
          "type": "AES",
          "privateKey": "0123456789abcdefhijklABCDEFHIJKL"
        },{
          "name": "hmac-key",
          "type": "HMAC",
          "privateKey": "ABCDEFHIJKLabcdefhijkl0123456789"
          }
        }]
      }
